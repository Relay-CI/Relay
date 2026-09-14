package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func gzipTarFixture(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestHandleSyncBundleAcceptsStreamingGzipTar(t *testing.T) {
	s := newPreviewPortTestServer(t)
	s.syncSessions = map[string]*SyncSession{}
	repo := t.TempDir()
	staging := t.TempDir()
	sess := &SyncSession{
		ID: "bundle-session", App: "demo", Env: EnvPreview, Branch: "main",
		RepoDir: repo, StagingDir: staging, CreatedAt: time.Now(), MaxBytes: 1 << 20,
	}
	s.syncSessions[sess.ID] = sess
	body := gzipTarFixture(t, map[string]string{"src/app.js": "export default 42;", "README.md": "hello"})
	req := httptest.NewRequest(http.MethodPut, "/api/sync/bundle/"+sess.ID, bytes.NewReader(body))
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	s.handleSyncBundle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("bundle status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response["files"] != float64(2) {
		t.Fatalf("expected two extracted files, got %#v", response)
	}
	got, err := os.ReadFile(filepath.Join(staging, "src", "app.js"))
	if err != nil || string(got) != "export default 42;" {
		t.Fatalf("unexpected staged file %q err=%v", got, err)
	}
}

func TestHandleSyncBundleRejectsUnknownEncoding(t *testing.T) {
	s := newPreviewPortTestServer(t)
	s.syncSessions = map[string]*SyncSession{}
	sess := &SyncSession{ID: "bad-encoding", StagingDir: t.TempDir(), MaxBytes: 1024}
	s.syncSessions[sess.ID] = sess
	req := httptest.NewRequest(http.MethodPut, "/api/sync/bundle/"+sess.ID, bytes.NewReader([]byte("data")))
	req.Header.Set("Content-Encoding", "br")
	rec := httptest.NewRecorder()
	s.handleSyncBundle(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleSyncBundleRejectsOversizedIgnoredEntryBeforeExtraction(t *testing.T) {
	s := newPreviewPortTestServer(t)
	s.syncSessions = map[string]*SyncSession{}
	sess := &SyncSession{ID: "oversized-ignored", StagingDir: t.TempDir(), MaxBytes: 1024}
	s.syncSessions[sess.ID] = sess
	body := gzipTarFixture(t, map[string]string{".env": strings.Repeat("x", 2048)})
	req := httptest.NewRequest(http.MethodPut, "/api/sync/bundle/"+sess.ID, bytes.NewReader(body))
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	s.handleSyncBundle(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleSyncPlanSkipsHashWhenSizeAndMtimeMatch(t *testing.T) {
	s := newPreviewPortTestServer(t)
	s.syncSessions = map[string]*SyncSession{}
	repo := t.TempDir()
	file := filepath.Join(repo, "app.js")
	if err := os.WriteFile(file, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	mtime := time.Unix(1_700_000_000, 0)
	if err := os.Chtimes(file, mtime, mtime); err != nil {
		t.Fatal(err)
	}
	sess := &SyncSession{ID: "plan-fast-path", RepoDir: repo, StagingDir: t.TempDir(), MaxBytes: 1024}
	s.syncSessions[sess.ID] = sess
	payload, _ := json.Marshal(SyncPlanRequest{Files: []ManifestFile{{
		Path: "app.js", Size: 4, Mtime: mtime.UnixMilli(), HashAlgo: "deliberately-unsupported", Hash: "unused",
	}}})
	req := httptest.NewRequest(http.MethodPost, "/api/sync/plan/"+sess.ID, bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	s.handleSyncPlan(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var plan SyncPlanResponse
	if err := json.NewDecoder(rec.Body).Decode(&plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Need) != 0 {
		t.Fatalf("metadata match should avoid hashing and uploading, need=%v", plan.Need)
	}
}
