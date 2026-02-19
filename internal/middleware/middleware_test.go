package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAuthMiddleware_NoKey(t *testing.T) {
	os.Unsetenv("CORTEX_API_KEY")
	handler := AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when no key set, got %d", w.Code)
	}
}

func TestAuthMiddleware_WithKey(t *testing.T) {
	os.Setenv("CORTEX_API_KEY", "secret")
	defer os.Unsetenv("CORTEX_API_KEY")
	handler := AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	t.Run("X-API-Key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "secret")
		w := httptest.NewRecorder()
		handler(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200 with X-API-Key, got %d", w.Code)
		}
	})
	t.Run("Authorization Bearer", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer secret")
		w := httptest.NewRecorder()
		handler(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200 with Bearer, got %d", w.Code)
		}
	})
	t.Run("missing header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		handler(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 without key, got %d", w.Code)
		}
	})
	t.Run("wrong key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "wrong")
		w := httptest.NewRecorder()
		handler(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 with wrong key, got %d", w.Code)
		}
	})
}
