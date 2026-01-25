package ingest

import (
	"net/http"

	"bookapi/internal/httpx"
)

type HTTPHandler struct {
	svc    *Service
	secret string
}

func NewHTTPHandler(svc *Service, secret string) *HTTPHandler {
	return &HTTPHandler{svc: svc, secret: secret}
}

// Ingest handles POST /internal/jobs/ingest
// @Summary Trigger catalog ingestion
// @Description Trigger ingestion job to populate catalog from Open Library
// @Tags internal
// @Accept json
// @Produce json
// @Param X-Internal-Secret header string true "Internal secret for authentication"
// @Success 200 {object} httpx.SuccessResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /internal/jobs/ingest [post]
func (h *HTTPHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	secret := r.Header.Get("X-Internal-Secret")
	if h.secret != "" && secret != h.secret {
		httpx.JSONError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid internal secret", nil)
		return
	}

	if err := h.svc.Run(r.Context()); err != nil {
		httpx.JSONError(w, r, http.StatusInternalServerError, "INGEST_FAILED", err.Error(), nil)
		return
	}

	httpx.JSONSuccess(w, r, map[string]string{"message": "ingestion completed"}, nil)
}
