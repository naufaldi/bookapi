package httpx

import (
	"log"
	"net/http"
	"runtime/debug"
)

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				requestID := RequestIDFrom(r)
				log.Printf("panic recovered: request_id=%s error=%v stack=%s", requestID, err, string(debug.Stack()))

				var wroteHeader bool
				if rw, ok := w.(*responseWriter); ok {
					wroteHeader = rw.wroteHeader()
				}

				if !wroteHeader {
					JSONError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}
