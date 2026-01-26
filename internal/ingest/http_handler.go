package ingest

import (
	"context"
	"log"
	"net/http"
	"time"

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
// @Success 202 {object} httpx.SuccessResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /internal/jobs/ingest [post]
func (h *HTTPHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	secret := r.Header.Get("X-Internal-Secret")
	if h.secret != "" && secret != h.secret {
		httpx.JSONError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid internal secret", nil)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	go func() {
		defer cancel()
		if err := h.svc.Run(ctx); err != nil {
			log.Printf("ingest job failed: %v", err)
		}
	}()

	httpx.JSONSuccessAccepted(w, r, map[string]string{"message": "ingestion started"}, nil)
}
