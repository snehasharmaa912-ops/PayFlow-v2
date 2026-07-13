package payflow

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type APIKeyAuth struct {
	keys map[string]bool
}

func NewAPIKeyAuth(keys []string) *APIKeyAuth {
	set := make(map[string]bool, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			set[k] = true
		}
	}
	return &APIKeyAuth{keys: set}
}

func (a *APIKeyAuth) Enabled() bool {
	return len(a.keys) > 0
}

func (a *APIKeyAuth) valid(key string) bool {
	return a.keys[key]
}

func (a *APIKeyAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.Enabled() || r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if key == "" || !a.valid(key) {
			writeError(w, http.StatusUnauthorized, "missing or invalid API key")
			return
		}

		next.ServeHTTP(w, r)
	})
}

type visitor struct {
	tokens     float64
	lastRefill time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     float64
	burst    float64
}

func NewRateLimiter(requestsPerSecond float64, burst float64) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     requestsPerSecond,
		burst:    burst,
	}
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[key]
	if !ok {
		v = &visitor{tokens: rl.burst, lastRefill: time.Now()}
		rl.visitors[key] = v
	}

	now := time.Now()
	elapsed := now.Sub(v.lastRefill).Seconds()
	v.tokens += elapsed * rl.rate
	if v.tokens > rl.burst {
		v.tokens = rl.burst
	}
	v.lastRefill = now

	if v.tokens < 1 {
		return false
	}
	v.tokens--
	return true
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if key == "" {
			key = r.RemoteAddr
		}

		if !rl.allow(key) {
			writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}
