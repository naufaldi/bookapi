package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"bookapi/internal/auth"
	"bookapi/internal/entity"

	"github.com/golang-jwt/jwt/v5"
)

// TestUser is a mock user for testing
var TestUser = entity.User{
	ID:        "test-user-id-123",
	Username:  "testuser",
	Email:     "test@example.com",
	Password:  "hashedpassword",
	Role:      "USER",
	CreatedAt: time.Now(),
	UpdatedAt: time.Now(),
}

// TestAdminUser is a mock admin user for testing
var TestAdminUser = entity.User{
	ID:        "test-admin-id-456",
	Username:  "adminuser",
	Email:     "admin@example.com",
	Password:  "hashedpassword",
	Role:      "ADMIN",
	CreatedAt: time.Now(),
	UpdatedAt: time.Now(),
}

// TestBook is a mock book for testing
var TestBook = entity.Book{
	ID:          "test-book-id-789",
	ISBN:        "978-0-123456-78-9",
	Title:       "Test Book Title",
	Genre:       "Fiction",
	Publisher:   "Test Publisher",
	Description: "A test book description",
	CreatedAt:   time.Now(),
	UpdatedAt:   time.Now(),
}

// GenerateTestToken generates a JWT token for testing
func GenerateTestToken(secret, userID, role string) string {
	token, _, _ := auth.GenerateToken(secret, userID, role, time.Hour)
	return token
}

// GenerateExpiredToken generates an expired JWT token for testing
func GenerateExpiredToken(secret, userID, role string) string {
	c := auth.Claims{
		Sub:  userID,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	token, _ := t.SignedString([]byte(secret))
	return token
}

// NewRequest creates a new HTTP request for testing
func NewRequest(method, path string, body interface{}) *http.Request {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	var r *http.Request
	if bodyBytes != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	return r
}

// NewRequestWithAuth creates a new HTTP request with JWT auth for testing
func NewRequestWithAuth(method, path string, body interface{}, token string) *http.Request {
	r := NewRequest(method, path, body)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return r
}

// RecordResponse records the HTTP response for testing
type RecordResponse struct {
	Code   int
	Header http.Header
	Body   map[string]interface{}
}

// RecordHTTPResponse records the HTTP response
func RecordHTTPResponse(w *httptest.ResponseRecorder) RecordResponse {
	result := w.Result()
	defer result.Body.Close()

	bodyBytes, _ := io.ReadAll(result.Body)

	var bodyMap map[string]interface{}
	if len(bodyBytes) > 0 {
		json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&bodyMap)
	}

	return RecordResponse{
		Code:   result.StatusCode,
		Header: result.Header,
		Body:   bodyMap,
	}
}

// AssertResponseCode checks if the response code matches expected
func AssertResponseCode(t interface {
	Errorf(format string, args ...any)
}, got, want int) {
	if got != want {
		t.Errorf("got status code %d, want %d", got, want)
	}
}

// AssertResponseBody checks if the response body contains expected field
func AssertResponseBody(t interface {
	Errorf(format string, args ...any)
}, body map[string]interface{}, key string, expectedValue interface{}) {
	value, ok := body[key]
	if !ok {
		t.Errorf("response body missing key %q", key)
		return
	}
	if value != expectedValue {
		t.Errorf("got %q for key %q, want %q", value, key, expectedValue)
	}
}
