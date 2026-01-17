package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestSizeLimitMiddleware_UnderLimit(t *testing.T) {
	maxBytes := int64(1024)
	middleware := RequestSizeLimitMiddleware(maxBytes)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := bytes.NewBuffer(make([]byte, 512))
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for request under limit, got %d", w.Code)
	}
}

func TestRequestSizeLimitMiddleware_OverLimit(t *testing.T) {
	maxBytes := int64(1024)
	middleware := RequestSizeLimitMiddleware(maxBytes)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := bytes.NewBuffer(make([]byte, 2048))
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected 413 for request over limit, got %d", w.Code)
	}
}

func TestRequestSizeLimitMiddleware_DifferentLimits(t *testing.T) {
	smallLimit := int64(100)
	largeLimit := int64(10000)

	smallMiddleware := RequestSizeLimitMiddleware(smallLimit)
	largeMiddleware := RequestSizeLimitMiddleware(largeLimit)

	smallHandler := smallMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	largeHandler := largeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := bytes.NewBuffer(make([]byte, 500))

	req1 := httptest.NewRequest(http.MethodPost, "/test", body)
	w1 := httptest.NewRecorder()
	smallHandler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected 413 for small limit, got %d", w1.Code)
	}

	body2 := bytes.NewBuffer(make([]byte, 500))
	req2 := httptest.NewRequest(http.MethodPost, "/test", body2)
	w2 := httptest.NewRecorder()
	largeHandler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected 200 for large limit, got %d", w2.Code)
	}
}
