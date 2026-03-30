package gateway

import (
	"net/http/httptest"
	"testing"
)

func TestSecurityConfigAuthenticatorLocal(t *testing.T) {
	cfg := SecurityConfig{Mode: SecurityModeLocal}
	req := httptest.NewRequest("GET", "http://localhost/api/v1/chat", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Origin", "http://localhost:5173")

	userID, ok := cfg.Authenticator()(req)
	if !ok {
		t.Fatal("expected local request to authenticate")
	}
	if userID != "local" {
		t.Fatalf("expected local user id, got %q", userID)
	}
}

func TestSecurityConfigAuthenticatorRejectsRemoteOriginInLocalMode(t *testing.T) {
	cfg := SecurityConfig{Mode: SecurityModeLocal}
	req := httptest.NewRequest("GET", "http://localhost/api/v1/chat", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Origin", "https://example.com")

	if _, ok := cfg.Authenticator()(req); ok {
		t.Fatal("expected remote origin to be rejected in local mode")
	}
}

func TestSecurityConfigAuthenticatorAPIKey(t *testing.T) {
	cfg := SecurityConfig{Mode: SecurityModeAPIKey, APIKey: "secret"}
	req := httptest.NewRequest("GET", "http://localhost/api/v1/chat", nil)
	req.RemoteAddr = "10.0.0.5:12345"
	req.Header.Set("X-API-Key", "secret")

	userID, ok := cfg.Authenticator()(req)
	if !ok {
		t.Fatal("expected apikey request to authenticate")
	}
	if userID != "api_key" {
		t.Fatalf("expected api_key user id, got %q", userID)
	}
}

func TestSecurityConfigAllowsRequest(t *testing.T) {
	tests := []struct {
		name    string
		cfg     SecurityConfig
		origin  string
		remote  string
		allowed bool
	}{
		{
			name:    "local request allowed",
			cfg:     SecurityConfig{Mode: SecurityModeLocal},
			origin:  "http://127.0.0.1:3000",
			remote:  "127.0.0.1:16789",
			allowed: true,
		},
		{
			name:    "remote host rejected in local mode",
			cfg:     SecurityConfig{Mode: SecurityModeLocal},
			origin:  "http://127.0.0.1:3000",
			remote:  "10.0.0.8:16789",
			allowed: false,
		},
		{
			name:    "open mode allows request",
			cfg:     SecurityConfig{Mode: SecurityModeOpen},
			origin:  "https://example.com",
			remote:  "10.0.0.8:16789",
			allowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://localhost/api/v1/chat", nil)
			req.RemoteAddr = tt.remote
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if got := tt.cfg.AllowsRequest(req); got != tt.allowed {
				t.Fatalf("AllowsRequest() = %v, want %v", got, tt.allowed)
			}
		})
	}
}
