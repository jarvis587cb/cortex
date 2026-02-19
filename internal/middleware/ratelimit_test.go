package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestRateLimitMiddleware(t *testing.T) {
	// Set rate limit for testing
	os.Setenv("CORTEX_RATE_LIMIT", "5")
	os.Setenv("CORTEX_RATE_LIMIT_WINDOW", "1s")
	defer os.Unsetenv("CORTEX_RATE_LIMIT")
	defer os.Unsetenv("CORTEX_RATE_LIMIT_WINDOW")

	handler := RateLimitMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	// First 5 requests should succeed
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		handler(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d should succeed, got status %d", i+1, w.Code)
		}
	}

	// 6th request should be rate limited
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("6th request should be rate limited, got status %d", w.Code)
	}

	// Check Retry-After header
	if w.Header().Get("Retry-After") == "" {
		t.Error("Retry-After header should be set")
	}
}

func TestRateLimitDisabled(t *testing.T) {
	// Disable rate limiting
	os.Setenv("CORTEX_RATE_LIMIT", "0")
	defer os.Unsetenv("CORTEX_RATE_LIMIT")

	handler := RateLimitMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("rate limiting should be disabled, got status %d", w.Code)
	}
}

func TestRateLimiterAllow(t *testing.T) {
	limiter := NewRateLimiter(10, 1*time.Second)

	// First 10 requests should be allowed
	for i := 0; i < 10; i++ {
		if !limiter.Allow("client1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 11th request should be denied
	if limiter.Allow("client1") {
		t.Error("11th request should be denied")
	}

	// Different client should still be allowed
	if !limiter.Allow("client2") {
		t.Error("different client should be allowed")
	}
}
