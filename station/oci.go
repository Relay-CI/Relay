package main

// OCI image puller — downloads base images from Docker Hub (or any OCI registry)
// without Docker. Used by the Dockerfile builder to provide FROM base layers.
//
// Flow:
//  1. GET token from auth.docker.io (public images need an anonymous bearer token)
//  2. GET /v2/<name>/manifests/<tag>  → manifest JSON
//  3. For each layer digest, GET /v2/<name>/blobs/<digest> → gzipped tar
//  4. Extract each layer in order, applying whiteout (.wh.) deletions
//  5. Save the final rootfs to a cache directory so repeated builds are instant
//
// Supported registries: Docker Hub (docker.io), ghcr.io, any OCI-compliant registry.
// Multi-arch manifests: automatically selects linux/amd64.

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ─── types ────────────────────────────────────────────────────────────────────

type ociManifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	// v2 schema 2
	Config *ociDescriptor   `json:"config,omitempty"`
	Layers []*ociDescriptor `json:"layers,omitempty"`
	// OCI image index (multi-arch)
	Manifests []*ociIndexEntry `json:"manifests,omitempty"`
}

type ociDescriptor struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

type ociIndexEntry struct {
	MediaType string      `json:"mediaType"`
	Size      int64       `json:"size"`
	Digest    string      `json:"digest"`
	Platform  ociPlatform `json:"platform"`
}

type ociPlatform struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
}

type ociConfig struct {
	Config struct {
		Cmd        []string          `json:"Cmd"`
		Entrypoint []string          `json:"Entrypoint"`
		Env        []string          `json:"Env"`
		ExposedPorts map[string]struct{} `json:"ExposedPorts"`
		WorkingDir string            `json:"WorkingDir"`
	} `json:"config"`
}

// ─── cache ────────────────────────────────────────────────────────────────────

func ociCacheDir() string {
	return filepath.Join(stateBaseDir(), "oci-cache")
}

// ociImageCacheDir returns the rootfs cache dir for a parsed image ref.
// e.g. "node:22" → .../oci-cache/node__22/rootfs
func ociImageCacheDir(image string) string {
	safe := strings.NewReplacer(":", "__", "/", "_").Replace(image)
	return filepath.Join(ociCacheDir(), safe)
}

func ociRootfsDir(image string) string {
	return filepath.Join(ociImageCacheDir(image), "rootfs")
}

func ociManifestCacheFile(image string) string {
	return filepath.Join(ociImageCacheDir(image), "manifest.json")
}

func ociConfigCacheFile(image string) string {
	return filepath.Join(ociImageCacheDir(image), "config.json")
}

// ociImageCached returns true if the image rootfs is already on disk.
func ociImageCached(image string) bool {
	_, err := os.Stat(ociRootfsDir(image))
	return err == nil
}

// ─── registry client ─────────────────────────────────────────────────────────

type ociClient struct {
	registry string // e.g. "registry-1.docker.io"
	repo     string // e.g. "library/node"
	tag      string // e.g. "22"
	token    string
	http     *http.Client
}

// newOCIClient parses an image reference and creates an authenticated client.
func newOCIClient(image string) (*ociClient, error) {
	registry, repo, tag := parseImageRef(image)
	c := &ociClient{
		registry: registry,
		repo:     repo,
		tag:      tag,
		http:     &http.Client{Timeout: 5 * time.Minute},
	}
	if err := c.fetchToken(); err != nil {
		return nil, fmt.Errorf("auth %s: %w", image, err)
	}
	return c, nil
}

// parseImageRef splits "node:22", "ubuntu:22.04", "ghcr.io/foo/bar:latest" etc.
func parseImageRef(image string) (registry, repo, tag string) {
	tag = "latest"
	if idx := strings.LastIndex(image, ":"); idx > 0 && !strings.Contains(image[idx:], "/") {
		tag = image[idx+1:]
		image = image[:idx]
	}
	// Check if there's a registry host (contains a dot or colon before first slash)
	parts := strings.SplitN(image, "/", 2)
	if len(parts) == 2 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":")) {
		registry = parts[0]
		repo = parts[1]
	} else {
		registry = "registry-1.docker.io"
		if len(parts) == 1 {
			// Official image like "node", "python" → library/node
			repo = "library/" + image
		} else {
			repo = image
		}
	}
	return
}

// fetchToken gets an anonymous bearer token from Docker Hub auth service.
// For private registries without this auth endpoint the token stays empty
// and requests are sent unauthenticated (works for public ghcr.io images).
func (c *ociClient) fetchToken() error {
	if c.registry != "registry-1.docker.io" {
		// Try Docker-compatible token endpoint; skip if unavailable.
		return nil
	}
	authURL := fmt.Sprintf(
		"https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull",
		url.QueryEscape(c.repo),
	)
	resp, err := c.http.Get(authURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("token request: HTTP %d", resp.StatusCode)
	}
	var tok struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return err
	}
	c.token = tok.Token
	return nil
}

func (c *ociClient) get(path string, accepts ...string) (*http.Response, error) {
	u := fmt.Sprintf("https://%s%s", c.registry, path)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	for _, a := range accepts {
		req.Header.Add("Accept", a)
	}
	return c.http.Do(req)
}

// ─── pull ─────────────────────────────────────────────────────────────────────

// PullImage downloads the image and extracts it to the cache. Returns the rootfs
// directory and a BuildManifest with CMD/env/port from the image config.
// If the image is already cached, it returns immediately.
func PullImage(image string, logf func(string, ...any)) (rootfs string, manifest *BuildManifest, err error) {
	rootfs = ociRootfsDir(image)
	if ociImageCached(image) {
		logf("image %s: cached", image)
		manifest, err = loadManifest(rootfs)
		if err != nil {
			manifest = &BuildManifest{Env: map[string]string{}}
		}
		return rootfs, manifest, nil
	}

	logf("pulling %s ...", image)

	client, err := newOCIClient(image)
	if err != nil {
		return "", nil, err
	}

	mf, err := client.fetchManifest()
	if err != nil {
		return "", nil, fmt.Errorf("manifest: %w", err)
	}

	if err := os.MkdirAll(rootfs, 0755); err != nil {
		return "", nil, err
	}

	logf("image %s: %d layer(s)", image, len(mf.Layers))
	for i, layer := range mf.Layers {
		logf("  layer %d/%d  %s  %.1fMB", i+1, len(mf.Layers),
			layer.Digest[:19], float64(layer.Size)/(1024*1024))
		if err := client.extractLayer(layer.Digest, rootfs); err != nil {
			return "", nil, fmt.Errorf("layer %s: %w", layer.Digest[:12], err)
		}
	}

	// Parse image config for CMD/ENV/EXPOSE.
	manifest = &BuildManifest{Env: map[string]string{}}
	if mf.Config != nil {
		cfg, cfgErr := client.fetchConfig(mf.Config.Digest)
		if cfgErr == nil {
			manifest.Cmd = cfg.Config.Cmd
			manifest.Entrypoint = cfg.Config.Entrypoint
			manifest.WorkDir = cfg.Config.WorkingDir
			for _, e := range cfg.Config.Env {
				k, v := parseEnv(e)
				manifest.Env[k] = v
			}
			for port := range cfg.Config.ExposedPorts {
				fmt.Sscanf(strings.Split(port, "/")[0], "%d", &manifest.Port)
				break
			}
		}
	}

	// Save manifest alongside rootfs so cached loads work.
	_ = saveManifest(rootfs, manifest)
	// Save raw manifest for diagnostics.
	if raw, _ := json.MarshalIndent(mf, "", "  "); raw != nil {
		_ = os.WriteFile(ociManifestCacheFile(image), raw, 0644)
	}

	logf("image %s: ready at %s", image, rootfs)
	return rootfs, manifest, nil
}

// fetchManifest retrieves the image manifest. Handles multi-arch image indexes
// by selecting the linux/amd64 entry automatically.
func (c *ociClient) fetchManifest() (*ociManifest, error) {
	path := fmt.Sprintf("/v2/%s/manifests/%s", c.repo, c.tag)
	resp, err := c.get(path,
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.oci.image.index.v1+json",
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var mf ociManifest
	if err := json.NewDecoder(resp.Body).Decode(&mf); err != nil {
		return nil, err
	}

	// Multi-arch index: find linux/amd64 entry and recurse.
	if len(mf.Manifests) > 0 {
		for _, entry := range mf.Manifests {
			if entry.Platform.OS == "linux" && entry.Platform.Architecture == "amd64" {
				c.tag = entry.Digest
				return c.fetchManifest()
			}
		}
		return nil, fmt.Errorf("no linux/amd64 manifest in image index")
	}

	return &mf, nil
}

// fetchConfig retrieves and parses the image config blob.
func (c *ociClient) fetchConfig(digest string) (*ociConfig, error) {
	path := fmt.Sprintf("/v2/%s/blobs/%s", c.repo, digest)
	resp, err := c.get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("config blob: HTTP %d", resp.StatusCode)
	}
	var cfg ociConfig
	return &cfg, json.NewDecoder(resp.Body).Decode(&cfg)
}

// extractLayer downloads a layer blob and extracts it into destDir.
// Supports gzip-compressed tar (the standard) and uncompressed tar.
// Applies OCI whiteout semantics: files named .wh.<name> delete <name>.
func (c *ociClient) extractLayer(digest, destDir string) error {
	// Check per-layer cache first to avoid re-downloading on re-pull.
	layerCacheFile := filepath.Join(ociCacheDir(), "layers", digest[7:]) // strip "sha256:"
	if _, err := os.Stat(layerCacheFile); err != nil {
		if err := c.downloadLayer(digest, layerCacheFile); err != nil {
			return err
		}
	}
	return applyLayer(layerCacheFile, destDir)
}

func (c *ociClient) downloadLayer(digest, destFile string) error {
	if err := os.MkdirAll(filepath.Dir(destFile), 0755); err != nil {
		return err
	}
	path := fmt.Sprintf("/v2/%s/blobs/%s", c.repo, digest)
	resp, err := c.get(path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("blob %s: HTTP %d", digest[:12], resp.StatusCode)
	}

	tmp := destFile + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	h := sha256.New()
	_, err = io.Copy(f, io.TeeReader(resp.Body, h))
	f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}

	// Verify digest.
	got := "sha256:" + hex.EncodeToString(h.Sum(nil))
	if got != digest {
		_ = os.Remove(tmp)
		return fmt.Errorf("digest mismatch: want %s got %s", digest[:12], got[:12])
	}
	return os.Rename(tmp, destFile)
}

// applyLayer extracts a gzip-compressed tar layer into destDir, applying
// OCI whiteout rules (delete markers) and respecting symlinks.
func applyLayer(layerFile, destDir string) error {
	f, err := os.Open(layerFile)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		// Not gzipped — try raw tar.
		f.Seek(0, io.SeekStart)
		return applyTar(tar.NewReader(f), destDir)
	}
	defer gr.Close()
	return applyTar(tar.NewReader(gr), destDir)
}

func applyTar(tr *tar.Reader, destDir string) error {
	destDir, _ = filepath.Abs(destDir)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Normalise path and guard against escapes.
		name := filepath.Clean(hdr.Name)
		if name == "." {
			continue
		}
		target := filepath.Join(destDir, name)
		if !strings.HasPrefix(target, destDir+string(filepath.Separator)) {
			continue // path escape guard
		}

		base := filepath.Base(name)

		// OCI whiteout: .wh.<name> means delete <name> in lower layers.
		if strings.HasPrefix(base, ".wh.") {
			victim := filepath.Join(filepath.Dir(target), strings.TrimPrefix(base, ".wh."))
			_ = os.RemoveAll(victim)
			continue
		}
		// Opaque whiteout .wh..wh..opq means delete all children of the dir.
		if base == ".wh..wh..opq" {
			entries, _ := os.ReadDir(filepath.Dir(target))
			for _, e := range entries {
				_ = os.RemoveAll(filepath.Join(filepath.Dir(target), e.Name()))
			}
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(target, hdr.FileInfo().Mode()|0111)

		case tar.TypeReg, tar.TypeRegA:
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			_ = os.Remove(target) // overwrite existing
			wf, werr := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode())
			if werr != nil {
				continue
			}
			_, _ = io.Copy(wf, tr)
			wf.Close()

		case tar.TypeSymlink:
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			_ = os.Remove(target)
			_ = os.Symlink(hdr.Linkname, target)

		case tar.TypeLink:
			old := filepath.Join(destDir, filepath.Clean(hdr.Linkname))
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			_ = os.Remove(target)
			_ = os.Link(old, target)
		}
	}
	return nil
}

// ─── CLI commands ─────────────────────────────────────────────────────────────

// cmdImagePull pulls an image to the local cache.
// Usage: station image pull <image>
func cmdImagePull(image string) {
	_, _, err := PullImage(image, func(f string, a ...any) {
		fmt.Printf(f+"\n", a...)
	})
	if err != nil {
		die("pull %s: %v", image, err)
	}
}

// cmdImageList lists cached images.
func cmdImageList() {
	entries, err := os.ReadDir(ociCacheDir())
	if err != nil {
		fmt.Println("no cached images")
		return
	}
	fmt.Printf("%-35s  %s\n", "IMAGE", "ROOTFS")
	for _, e := range entries {
		if e.IsDir() && e.Name() != "layers" {
			name := strings.NewReplacer("__", ":", "_", "/").Replace(e.Name())
			rootfs := filepath.Join(ociCacheDir(), e.Name(), "rootfs")
			fmt.Printf("%-35s  %s\n", name, rootfs)
		}
	}
}

// cmdImageRemove deletes a cached image.
func cmdImageRemove(image string) {
	dir := ociImageCacheDir(image)
	if err := os.RemoveAll(dir); err != nil {
		die("remove: %v", err)
	}
	fmt.Printf("removed %s\n", image)
}

