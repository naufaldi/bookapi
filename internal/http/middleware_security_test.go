package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestSecurityHeadersMiddleware_HeadersSet(t *testing.T) {
	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":         "DENY",
		"X-XSS-Protection":        "1; mode=block",
		"Content-Security-Policy":  "default-src 'self'",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := w.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Expected %s header to be %s, got %s", header, expectedValue, actualValue)
		}
	}
}

func TestSecurityHeadersMiddleware_HSTSEnabled(t *testing.T) {
	os.Setenv("ENABLE_HSTS", "true")
	defer os.Unsetenv("ENABLE_HSTS")

	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	if hsts == "" {
		t.Error("Expected HSTS header when ENABLE_HSTS=true")
	}
	if hsts != "max-age=31536000; includeSubDomains" {
		t.Errorf("Expected HSTS header with correct value, got %s", hsts)
	}
}

func TestSecurityHeadersMiddleware_HSTSDisabled(t *testing.T) {
	os.Unsetenv("ENABLE_HSTS")

	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	if hsts != "" {
		t.Errorf("Expected no HSTS header when disabled, got %s", hsts)
	}
}
