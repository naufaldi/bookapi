package httpx

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
	Meta    interface{}       `json:"meta,omitempty"`
}

type ErrorResponseBody struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func buildMeta(r *http.Request, customMeta interface{}) interface{} {
	requestID := RequestIDFrom(r)
	if requestID == "" && customMeta == nil {
		return nil
	}
	meta := make(map[string]interface{})
	if requestID != "" {
		meta["request_id"] = requestID
	}
	if customMeta != nil {
		if customMap, ok := customMeta.(map[string]interface{}); ok {
			for k, v := range customMap {
				meta[k] = v
			}
		} else {
			meta["custom"] = customMeta
		}
	}
	return meta
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

func JSONSuccessWithRequest(r *http.Request, w http.ResponseWriter, data interface{}, customMeta interface{}) {
	meta := buildMeta(r, customMeta)
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

func JSONSuccessCreatedWithRequest(r *http.Request, w http.ResponseWriter, data interface{}) {
	meta := buildMeta(r, nil)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
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

func JSONErrorWithRequest(r *http.Request, w http.ResponseWriter, statusCode int, code string, message string, details []ErrorDetail) {
	meta := buildMeta(r, nil)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Success: false,
		Error: ErrorResponseBody{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: meta,
	})
}
