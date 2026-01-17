package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}
	meta := map[string]int{"total": 10}

	JSONSuccess(w, data, meta)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type application/json")
	}

	var response SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}

	if response.Data == nil {
		t.Error("Expected data to be present")
	}
}

func TestJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	code := "VALIDATION_ERROR"
	message := "Invalid input"
	details := []ErrorDetail{
		{Field: "email", Message: "email is required"},
	}

	JSONError(w, http.StatusBadRequest, code, message, details)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type application/json")
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success to be false")
	}

	if response.Error.Code != code {
		t.Errorf("Expected error code %s, got %s", code, response.Error.Code)
	}

	if len(response.Error.Details) != 1 {
		t.Errorf("Expected 1 error detail, got %d", len(response.Error.Details))
	}
}

func TestJSONSuccessNoContent(t *testing.T) {
	w := httptest.NewRecorder()

	JSONSuccessNoContent(w)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}
}
