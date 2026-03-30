package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type APIProxy struct {
	targetBase string
	client     *http.Client
	mu         sync.RWMutex
}

var apiProxy *APIProxy
var apiProxyOnce sync.Once

func GetAPIProxy() *APIProxy {
	apiProxyOnce.Do(func() {
		apiProxy = &APIProxy{
			targetBase: "http://localhost:16789",
			client: &http.Client{
				Transport: &http.Transport{
					MaxIdleConns:        100,
					MaxIdleConnsPerHost: 10,
				},
			},
		}
	})
	return apiProxy
}

func (p *APIProxy) SetTargetBase(base string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.targetBase = strings.TrimSuffix(base, "/")
}

func (p *APIProxy) GetTargetBase() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.targetBase
}

func (p *APIProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if strings.HasPrefix(path, "/proxy/api/") {
		p.handleAPIProxy(w, r)
		return
	}

	if strings.HasPrefix(path, "/proxy/ws/") {
		p.handleWSProxy(w, r)
		return
	}

	if path == "/proxy/config" && r.Method == "GET" {
		p.getConfig(w, r)
		return
	}

	if path == "/proxy/config" && r.Method == "POST" {
		p.setConfig(w, r)
		return
	}

	http.Error(w, "Not Found", http.StatusNotFound)
}

func (p *APIProxy) handleAPIProxy(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/proxy")
	targetURL := p.GetTargetBase() + path

	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	var body io.Reader
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		body = bytes.NewReader(bodyBytes)
	}

	proxyReq, err := http.NewRequest(r.Method, targetURL, body)
	if err != nil {
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}

	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	proxyReq.Header.Del("Origin")
	proxyReq.Header.Del("Referer")

	resp, err := p.client.Do(proxyReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Proxy request failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *APIProxy) handleWSProxy(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "WebSocket proxy not implemented", http.StatusNotImplemented)
}

func (p *APIProxy) getConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"apiBase": p.GetTargetBase(),
	})
}

func (p *APIProxy) setConfig(w http.ResponseWriter, r *http.Request) {
	var config struct {
		ApiBase string `json:"apiBase"`
	}

	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if config.ApiBase != "" {
		if _, err := url.Parse(config.ApiBase); err != nil {
			http.Error(w, "Invalid API base URL", http.StatusBadRequest)
			return
		}
		p.SetTargetBase(config.ApiBase)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"success": true,
	})
}
