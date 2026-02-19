package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// MethodAllowed wraps an HTTP handler to check if the request method is allowed
func MethodAllowed(handler http.HandlerFunc, methods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, method := range methods {
			if r.Method == method {
				handler(w, r)
				return
			}
		}
		slog.Warn("method not allowed", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// AuthMiddleware enforces optional API key authentication.
// If CORTEX_API_KEY is set, requests must send X-API-Key or Authorization: Bearer <key>.
// If CORTEX_API_KEY is empty, all requests are allowed (local/dev mode).
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	apiKey := os.Getenv("CORTEX_API_KEY")
	if apiKey == "" {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		provided := r.Header.Get("X-API-Key")
		if provided == "" {
			auth := r.Header.Get("Authorization")
			provided = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		}
		if provided != apiKey {
			slog.Warn("unauthorized", "path", r.URL.Path, "ip", r.RemoteAddr)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
