package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	AccessPolicyPublic      = "public"
	AccessPolicyRelayLogin  = "relay-login"
	AccessPolicySignedLink  = "signed-link"
	AccessPolicyIPAllowlist = "ip-allowlist"
)

func normalizeAccessPolicy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AccessPolicyPublic:
		return AccessPolicyPublic
	case "":
		return ""
	case "relay-login", "relay_login", "login", "auth":
		return AccessPolicyRelayLogin
	case "signed-link", "signed_link", "signed":
		return AccessPolicySignedLink
	case "ip-allowlist", "ip_allowlist", "allowlist":
		return AccessPolicyIPAllowlist
	default:
		return ""
	}
}

func normalizeIPAllowlist(raw string) string {
	lines := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ';'
	})
	out := make([]string, 0, len(lines))
	seen := map[string]struct{}{}
	for _, line := range lines {
		entry := strings.TrimSpace(line)
		if entry == "" {
			continue
		}
		if ip := net.ParseIP(entry); ip == nil {
			if _, _, err := net.ParseCIDR(entry); err != nil {
				continue
			}
		}
		if _, ok := seen[entry]; ok {
			continue
		}
		seen[entry] = struct{}{}
		out = append(out, entry)
	}
	return strings.Join(out, "\n")
}

func requestRemoteIP(r *http.Request) net.IP {
	candidates := []string{
		r.Header.Get("X-Forwarded-For"),
		r.Header.Get("X-Real-Ip"),
		r.RemoteAddr,
	}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if strings.Contains(candidate, ",") {
			candidate = strings.TrimSpace(strings.Split(candidate, ",")[0])
		}
		if host, _, err := net.SplitHostPort(candidate); err == nil {
			candidate = host
		}
		if ip := net.ParseIP(candidate); ip != nil {
			return ip
		}
	}
	return nil
}

func ipAllowedByList(raw string, ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, entry := range strings.Split(normalizeIPAllowlist(raw), "\n") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if cidrIP, cidr, err := net.ParseCIDR(entry); err == nil {
			if cidr.Contains(ip) || ip.Equal(cidrIP) {
				return true
			}
			continue
		}
		if allowed := net.ParseIP(entry); allowed != nil && allowed.Equal(ip) {
			return true
		}
	}
	return false
}

func (s *Server) edgeSigningSecret(app string, env DeployEnv, branch string) string {
	base := strings.TrimSpace(os.Getenv("RELAY_EDGE_SIGNING_SECRET"))
	if base == "" {
		base = strings.TrimSpace(os.Getenv("RELAY_SECRET_KEY"))
	}
	if base == "" {
		base = strings.TrimSpace(s.apiToken)
	}
	if base == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(base))
	_, _ = mac.Write([]byte(app))
	_, _ = mac.Write([]byte{'\n'})
	_, _ = mac.Write([]byte(env))
	_, _ = mac.Write([]byte{'\n'})
	_, _ = mac.Write([]byte(branch))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Server) verifySignedEdgeLink(r *http.Request, app string, env DeployEnv, branch string) bool {
	secret := strings.TrimSpace(s.edgeSigningSecret(app, env, branch))
	if secret == "" {
		return false
	}
	expValue := strings.TrimSpace(r.URL.Query().Get("relay_exp"))
	sig := strings.TrimSpace(r.URL.Query().Get("relay_sig"))
	if expValue == "" || sig == "" {
		return false
	}
	exp, err := strconv.ParseInt(expValue, 10, 64)
	if err != nil {
		return false
	}
	if time.Now().Unix() > exp {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(app))
	_, _ = mac.Write([]byte{'\n'})
	_, _ = mac.Write([]byte(env))
	_, _ = mac.Write([]byte{'\n'})
	_, _ = mac.Write([]byte(branch))
	_, _ = mac.Write([]byte{'\n'})
	_, _ = mac.Write([]byte(expValue))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}

func sharedCookieDomain(hosts ...string) string {
	var suffix []string
	for _, host := range hosts {
		host = normalizedHostname(host)
		if host == "" || net.ParseIP(host) != nil {
			continue
		}
		parts := strings.Split(host, ".")
		if len(parts) < 2 {
			continue
		}
		if suffix == nil {
			suffix = parts
			continue
		}
		i := len(suffix) - 1
		j := len(parts) - 1
		matched := []string{}
		for i >= 0 && j >= 0 && strings.EqualFold(suffix[i], parts[j]) {
			matched = append([]string{suffix[i]}, matched...)
			i--
			j--
		}
		suffix = matched
	}
	if len(suffix) < 2 {
		return ""
	}
	return strings.Join(suffix, ".")
}

func (s *Server) sessionCookieDomainForRequest(r *http.Request) string {
	override := normalizedHostname(os.Getenv("RELAY_SESSION_COOKIE_DOMAIN"))
	if override != "" {
		host := normalizedRequestHost(r)
		if host == override || strings.HasSuffix(host, "."+override) {
			return override
		}
		return ""
	}
	domain := sharedCookieDomain(s.serverDashboardHost(), s.serverBaseDomain())
	if domain == "" {
		return ""
	}
	host := normalizedRequestHost(r)
	if host == domain || strings.HasSuffix(host, "."+domain) {
		return domain
	}
	return ""
}

func (s *Server) requestHasRelayIdentity(r *http.Request) bool {
	if s.hasUsers() {
		return s.validateUserSession(r) != nil
	}
	token, _ := s.requestToken(r)
	return s.validateToken(token)
}

func (s *Server) authorizeEdgeAccess(r *http.Request, app string, env DeployEnv, branch string, accessPolicy string, ipAllowlist string) (int, string) {
	switch firstNonEmpty(normalizeAccessPolicy(accessPolicy), s.lanePolicy(env).DefaultAccessPolicy) {
	case AccessPolicyPublic:
		return http.StatusNoContent, ""
	case AccessPolicyRelayLogin:
		if s.requestHasRelayIdentity(r) {
			return http.StatusNoContent, ""
		}
		return http.StatusUnauthorized, "relay login required"
	case AccessPolicySignedLink:
		if s.verifySignedEdgeLink(r, app, env, branch) {
			return http.StatusNoContent, ""
		}
		return http.StatusUnauthorized, "signed link required"
	case AccessPolicyIPAllowlist:
		if ipAllowedByList(ipAllowlist, requestRemoteIP(r)) {
			return http.StatusNoContent, ""
		}
		return http.StatusForbidden, "ip not allowed"
	default:
		return http.StatusForbidden, "unsupported access policy"
	}
}

func (s *Server) handleEdgeAuthz(w http.ResponseWriter, r *http.Request) {
	app := strings.TrimSpace(r.URL.Query().Get("app"))
	env := normalizeDeployEnv(r.URL.Query().Get("env"))
	branch := strings.TrimSpace(r.URL.Query().Get("branch"))
	if app == "" || env == "" || branch == "" {
		httpError(w, 400, "app, env, branch required")
		return
	}
	state, _ := s.getAppState(app, env, branch)
	accessPolicy := ""
	ipAllowlist := ""
	if state != nil {
		accessPolicy = state.AccessPolicy
		ipAllowlist = state.IPAllowlist
	}
	status, msg := s.authorizeEdgeAccess(r, app, env, branch, accessPolicy, ipAllowlist)
	if status != http.StatusNoContent {
		httpError(w, status, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func edgeAuthProxyURL(relayPort int, app string, env DeployEnv, branch string, host string) string {
	if relayPort <= 0 || strings.TrimSpace(host) == "" {
		return ""
	}
	return fmt.Sprintf("http://%s:%d/api/edge/authz?app=%s&env=%s&branch=%s",
		host,
		relayPort,
		url.QueryEscape(app),
		url.QueryEscape(string(env)),
		url.QueryEscape(branch),
	)
}
