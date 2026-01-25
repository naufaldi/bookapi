package httpx

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type rateLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimitMiddleware struct {
	limiters map[string]*rateLimiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	cleanup  time.Duration
}

func NewRateLimitMiddleware(rps float64, burst int) *RateLimitMiddleware {
	rl := &RateLimitMiddleware{
		limiters: make(map[string]*rateLimiter),
		rate:     rate.Limit(rps),
		burst:    burst,
		cleanup:  5 * time.Minute,
	}

	go rl.cleanupLimiters()
	return rl
}

func (rl *RateLimitMiddleware) cleanupLimiters() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for key, limiter := range rl.limiters {
			if time.Since(limiter.lastSeen) > rl.cleanup {
				delete(rl.limiters, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimitMiddleware) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = &rateLimiter{
			limiter:  rate.NewLimiter(rl.rate, rl.burst),
			lastSeen: time.Now(),
		}
		rl.limiters[key] = limiter
	} else {
		limiter.lastSeen = time.Now()
	}

	return limiter.limiter
}

func (rl *RateLimitMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			key = forwarded
		}

		limiter := rl.getLimiter(key)
		if !limiter.Allow() {
			JSONError(w, r, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many requests", nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}
