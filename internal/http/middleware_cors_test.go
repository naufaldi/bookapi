package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	allowedOrigins := []string{"http://localhost:3000", "http://localhost:5173"}
	middleware := CORSMiddleware(allowedOrigins)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("Expected CORS header for allowed origin, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("Expected credentials header, got %s", w.Header().Get("Access-Control-Allow-Credentials"))
	}
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	allowedOrigins := []string{"http://localhost:3000"}
	middleware := CORSMiddleware(allowedOrigins)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Expected no CORS header for disallowed origin, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddleware_OPTIONSRequest(t *testing.T) {
	allowedOrigins := []string{"http://localhost:3000"}
	middleware := CORSMiddleware(allowedOrigins)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected 204 for OPTIONS request, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected CORS methods header for OPTIONS request")
	}
}

func TestCORSMiddleware_HeadersAndMethods(t *testing.T) {
	allowedOrigins := []string{"http://localhost:3000"}
	middleware := CORSMiddleware(allowedOrigins)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	methods := w.Header().Get("Access-Control-Allow-Methods")
	if !strings.Contains(methods, "GET") || !strings.Contains(methods, "POST") {
		t.Errorf("Expected methods header to include GET and POST, got %s", methods)
	}

	headers := w.Header().Get("Access-Control-Allow-Headers")
	if !strings.Contains(headers, "Content-Type") || !strings.Contains(headers, "Authorization") {
		t.Errorf("Expected headers to include Content-Type and Authorization, got %s", headers)
	}
}
