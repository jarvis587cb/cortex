package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	mu          sync.RWMutex
	clients     map[string]*clientLimiter
	rate        int           // requests per window
	window      time.Duration // time window
	cleanupTick *time.Ticker
}

type clientLimiter struct {
	tokens     int
	lastAccess time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*clientLimiter),
		rate:    rate,
		window:  window,
	}

	// Cleanup old entries every minute
	rl.cleanupTick = time.NewTicker(1 * time.Minute)
	go rl.cleanup()

	return rl
}

// cleanup removes old client entries
func (rl *RateLimiter) cleanup() {
	for range rl.cleanupTick.C {
		rl.mu.Lock()
		now := time.Now()
		for key, cl := range rl.clients {
			cl.mu.Lock()
			if now.Sub(cl.lastAccess) > rl.window*2 {
				delete(rl.clients, key)
			}
			cl.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow(clientID string) bool {
	rl.mu.Lock()
	cl, exists := rl.clients[clientID]
	if !exists {
		cl = &clientLimiter{
			tokens:     rl.rate,
			lastAccess: time.Now(),
		}
		rl.clients[clientID] = cl
	}
	rl.mu.Unlock()

	cl.mu.Lock()
	defer cl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(cl.lastAccess)

	// Refill tokens based on elapsed time
	if elapsed >= rl.window {
		cl.tokens = rl.rate
	} else {
		// Refill proportionally
		refill := int(float64(rl.rate) * elapsed.Seconds() / rl.window.Seconds())
		if refill > 0 {
			cl.tokens = min(cl.tokens+refill, rl.rate)
		}
	}

	if cl.tokens > 0 {
		cl.tokens--
		cl.lastAccess = now
		return true
	}

	return false
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getClientID extracts client identifier from request
func getClientID(r *http.Request) string {
	// Try API key first (if available)
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		return authHeader
	}

	// Fallback to IP address
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	return ip
}

// RateLimitMiddleware provides rate limiting
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	// Get rate limit config from environment
	rateStr := os.Getenv("CORTEX_RATE_LIMIT")
	windowStr := os.Getenv("CORTEX_RATE_LIMIT_WINDOW")

	rate := 100 // Default: 100 requests
	window := 1 * time.Minute // Default: 1 minute

	if rateStr != "" {
		if r, err := strconv.Atoi(rateStr); err == nil && r > 0 {
			rate = r
		}
	}

	if windowStr != "" {
		if w, err := time.ParseDuration(windowStr); err == nil && w > 0 {
			window = w
		}
	}

	// If rate limiting is disabled (rate = 0), skip middleware
	if rate == 0 {
		return next
	}

	limiter := NewRateLimiter(rate, window)

	return func(w http.ResponseWriter, r *http.Request) {
		clientID := getClientID(r)

		if !limiter.Allow(clientID) {
			slog.Warn("rate limit exceeded", "client", clientID, "path", r.URL.Path)
			w.Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}
