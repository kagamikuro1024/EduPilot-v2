package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthz(t *testing.T) {
	tests := []struct {
		name, method string
		code         int
	}{
		{"GET → 200", http.MethodGet, http.StatusOK},
		{"POST → 405", http.MethodPost, http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			NewRouter().ServeHTTP(rec, httptest.NewRequest(tt.method, "/healthz", nil))
			if rec.Code != tt.code {
				t.Fatalf("code=%d want %d", rec.Code, tt.code)
			}
			if tt.code != http.StatusOK {
				return
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Fatalf("content-type=%q", ct)
			}
			if body := strings.TrimSpace(rec.Body.String()); body != `{"status":"ok"}` {
				t.Fatalf("body=%q", body)
			}
		})
	}
}
