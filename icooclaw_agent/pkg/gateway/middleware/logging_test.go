package middleware

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := RequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test?foo=bar", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("unmarshal log line: %v", err)
	}
	if line["msg"] != "request completed" {
		t.Fatalf("unexpected msg: %v", line["msg"])
	}
	if got := int(line["status"].(float64)); got != http.StatusCreated {
		t.Fatalf("unexpected status: %d", got)
	}
}

type hijackableRecorder struct {
	*httptest.ResponseRecorder
}

func (r *hijackableRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}

func TestResponseRecorderImplementsHijacker(t *testing.T) {
	rec := &responseRecorder{ResponseWriter: &hijackableRecorder{ResponseRecorder: httptest.NewRecorder()}}
	if _, ok := any(rec).(http.Hijacker); !ok {
		t.Fatal("responseRecorder should implement http.Hijacker")
	}

	_, _, err := rec.Hijack()
	if err != nil {
		t.Fatalf("Hijack() error = %v", err)
	}
}
