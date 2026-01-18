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

func (h *HTTPHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	secret := r.Header.Get("X-Internal-Secret")
	if h.secret != "" && secret != h.secret {
		httpx.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid internal secret", nil)
		return
	}

	if err := h.svc.Run(r.Context()); err != nil {
		httpx.JSONError(w, http.StatusInternalServerError, "INGEST_FAILED", err.Error(), nil)
		return
	}

	httpx.JSONSuccess(w, map[string]string{"message": "ingestion completed"}, nil)
}
