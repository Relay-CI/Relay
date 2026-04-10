package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type signedLinkRequest struct {
	App           string    `json:"app"`
	Env           DeployEnv `json:"env"`
	Branch        string    `json:"branch"`
	ExpiresInMins int       `json:"expires_in_minutes"`
}

type signedLinkResponse struct {
	URL          string `json:"url"`
	BaseURL      string `json:"base_url"`
	AccessPolicy string `json:"access_policy"`
	ExpiresAt    int64  `json:"expires_at"`
}

func requestActorLabel(s *Server, r *http.Request) string {
	if r != nil && r.Context().Value(ctxKeySocket) == true {
		return "socket"
	}
	if s != nil {
		if sess := s.validateUserSession(r); sess != nil {
			return sess.Username
		}
		if token, _ := s.requestToken(r); token != "" {
			return "token"
		}
	}
	return "system"
}

func requestActorRole(s *Server, r *http.Request) string {
	if r != nil && r.Context().Value(ctxKeySocket) == true {
		return "owner"
	}
	if s != nil {
		if sess := s.validateUserSession(r); sess != nil {
			return sess.Role
		}
		if token, _ := s.requestToken(r); token != "" {
			return "owner"
		}
	}
	return ""
}

func edgeSignedURL(baseURL string, app string, env DeployEnv, branch string, secret string, expiresAt int64) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" || strings.TrimSpace(secret) == "" || expiresAt <= 0 {
		return ""
	}
	expValue := strconv.FormatInt(expiresAt, 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(app))
	_, _ = mac.Write([]byte{'\n'})
	_, _ = mac.Write([]byte(env))
	_, _ = mac.Write([]byte{'\n'})
	_, _ = mac.Write([]byte(branch))
	_, _ = mac.Write([]byte{'\n'})
	_, _ = mac.Write([]byte(expValue))
	sig := hex.EncodeToString(mac.Sum(nil))

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	query := parsed.Query()
	query.Set("relay_exp", expValue)
	query.Set("relay_sig", sig)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func routeURLForRequest(r *http.Request, st *AppState) string {
	if st == nil {
		return ""
	}
	if host := strings.TrimSpace(st.PublicHost); host != "" {
		return "https://" + host
	}
	if st.HostPort <= 0 {
		return ""
	}
	host := normalizedRequestHost(r)
	if host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%d", host, st.HostPort)
}

func (s *Server) handleSignedLink(w http.ResponseWriter, r *http.Request) {
	var req signedLinkRequest
	switch r.Method {
	case http.MethodGet:
		req.App = strings.TrimSpace(r.URL.Query().Get("app"))
		req.Env = normalizeDeployEnv(r.URL.Query().Get("env"))
		req.Branch = strings.TrimSpace(r.URL.Query().Get("branch"))
		if mins, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("expires_in_minutes"))); err == nil {
			req.ExpiresInMins = mins
		}
	case http.MethodPost:
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, 400, "invalid json")
			return
		}
		req.Env = normalizeDeployEnv(string(req.Env))
	default:
		httpError(w, 405, "method not allowed")
		return
	}

	if !validDeployTarget(req.App, req.Env, req.Branch) {
		httpError(w, 400, "app, branch, env required")
		return
	}

	st, err := s.getAppState(req.App, req.Env, req.Branch)
	if err != nil || st == nil {
		httpError(w, 404, "app state not found")
		return
	}
	accessPolicy := firstNonEmpty(normalizeAccessPolicy(st.AccessPolicy), s.lanePolicy(req.Env).DefaultAccessPolicy)
	if accessPolicy != AccessPolicySignedLink {
		httpError(w, 400, "access_policy must be signed-link to generate share links")
		return
	}

	baseURL := routeURLForRequest(r, st)
	if baseURL == "" {
		httpError(w, 400, "this lane does not currently have a reachable route")
		return
	}

	minutes := req.ExpiresInMins
	if minutes <= 0 {
		minutes = 24 * 60
	}
	if minutes > 7*24*60 {
		minutes = 7 * 24 * 60
	}
	expiresAt := time.Now().Add(time.Duration(minutes) * time.Minute).Unix()
	secret := s.edgeSigningSecret(req.App, req.Env, req.Branch)
	if strings.TrimSpace(secret) == "" {
		httpError(w, 500, "edge signing secret is not configured")
		return
	}

	link := edgeSignedURL(baseURL, req.App, req.Env, req.Branch, secret, expiresAt)
	if link == "" {
		httpError(w, 500, "failed to generate signed link")
		return
	}

	actor := requestActorLabel(s, r)
	s.auditLog(actor, "signed_link.create", req.App, fmt.Sprintf("env=%s branch=%s expires_in_minutes=%d", req.Env, req.Branch, minutes))
	writeJSON(w, 200, signedLinkResponse{
		URL:          link,
		BaseURL:      baseURL,
		AccessPolicy: accessPolicy,
		ExpiresAt:    expiresAt,
	})
}
