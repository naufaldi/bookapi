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

func JSONSuccess(w http.ResponseWriter, r *http.Request, data interface{}, meta interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	requestID := RequestIDFrom(r)
	metaMap := make(map[string]interface{})
	if meta != nil {
		if m, ok := meta.(map[string]interface{}); ok {
			metaMap = m
		} else if m, ok := meta.(map[string]any); ok {
			metaMap = m
		}
	}
	if requestID != "" {
		metaMap["request_id"] = requestID
	}

	var finalMeta interface{}
	if len(metaMap) > 0 {
		finalMeta = metaMap
	} else if requestID != "" {
		finalMeta = map[string]string{"request_id": requestID}
	} else if meta != nil {
		finalMeta = meta
	}

	json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Data:    data,
		Meta:    finalMeta,
	})
}

func JSONSuccessCreated(w http.ResponseWriter, r *http.Request, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	requestID := RequestIDFrom(r)
	var meta interface{}
	if requestID != "" {
		meta = map[string]string{"request_id": requestID}
	}

	json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func JSONSuccessAccepted(w http.ResponseWriter, r *http.Request, data interface{}, meta interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	requestID := RequestIDFrom(r)
	metaMap := make(map[string]interface{})
	if meta != nil {
		if m, ok := meta.(map[string]interface{}); ok {
			metaMap = m
		} else if m, ok := meta.(map[string]any); ok {
			metaMap = m
		}
	}
	if requestID != "" {
		metaMap["request_id"] = requestID
	}

	var finalMeta interface{}
	if len(metaMap) > 0 {
		finalMeta = metaMap
	} else if requestID != "" {
		finalMeta = map[string]string{"request_id": requestID}
	} else if meta != nil {
		finalMeta = meta
	}

	json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Data:    data,
		Meta:    finalMeta,
	})
}

func JSONSuccessNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func JSONError(w http.ResponseWriter, r *http.Request, statusCode int, code string, message string, details []ErrorDetail) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	requestID := RequestIDFrom(r)
	var meta interface{}
	if requestID != "" {
		meta = map[string]string{"request_id": requestID}
	}

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
