package http

import (
	"encoding/json"
	"net/http"
)

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

type ErrorResponse struct {
	Success bool              `json:"success"`
	Error   ErrorResponseBody `json:"error"`
}

type ErrorResponseBody struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func JSONSuccess(w http.ResponseWriter, data interface{}, meta interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func JSONSuccessCreated(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Data:    data,
	})
}

func JSONSuccessNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func JSONError(w http.ResponseWriter, statusCode int, code string, message string, details []ErrorDetail) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Success: false,
		Error: ErrorResponseBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}
