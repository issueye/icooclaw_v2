package gateway

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

const (
	SecurityModeLocal  = "local"
	SecurityModeAPIKey = "apikey"
	SecurityModeOpen   = "open"
)

type SecurityConfig struct {
	Mode   string
	APIKey string
}

func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{Mode: SecurityModeLocal}
}

func (c SecurityConfig) Authenticator() func(r *http.Request) (string, bool) {
	mode := strings.ToLower(strings.TrimSpace(c.Mode))
	switch mode {
	case "", SecurityModeLocal:
		return authenticateLocalRequest
	case SecurityModeAPIKey:
		return func(r *http.Request) (string, bool) {
			apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
			if apiKey == "" {
				apiKey = strings.TrimSpace(r.URL.Query().Get("api_key"))
			}
			if apiKey == "" {
				authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
				if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
					apiKey = strings.TrimSpace(authHeader[7:])
				}
			}
			if apiKey == "" || apiKey != c.APIKey {
				return "", false
			}
			return "api_key", true
		}
	case SecurityModeOpen:
		return func(r *http.Request) (string, bool) {
			return "open", true
		}
	default:
		return func(r *http.Request) (string, bool) {
			return "", false
		}
	}
}

func (c SecurityConfig) AllowsRequest(r *http.Request) bool {
	mode := strings.ToLower(strings.TrimSpace(c.Mode))
	switch mode {
	case "", SecurityModeLocal:
		return isTrustedLocalOrigin(r.Header.Get("Origin")) && isLoopbackRequest(r.RemoteAddr)
	case SecurityModeAPIKey, SecurityModeOpen:
		return true
	default:
		return false
	}
}

func authenticateLocalRequest(r *http.Request) (string, bool) {
	if !isTrustedLocalOrigin(r.Header.Get("Origin")) {
		return "", false
	}
	if !isLoopbackRequest(r.RemoteAddr) {
		return "", false
	}
	return "local", true
}

func isTrustedLocalOrigin(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" || origin == "null" {
		return true
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}

	host := strings.ToLower(parsed.Hostname())
	switch host {
	case "localhost", "127.0.0.1", "::1", "[::1]", "wails.localhost":
		return true
	default:
		return false
	}
}

func isLoopbackRequest(remoteAddr string) bool {
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}

	host = strings.Trim(host, "[]")
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
