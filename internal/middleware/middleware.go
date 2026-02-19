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

// AuthMiddleware provides API key authentication
// If CORTEX_API_KEY is not set, authentication is disabled (for development)
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	apiKey := os.Getenv("CORTEX_API_KEY")

	// If no API key is set, skip authentication (development mode)
	if apiKey == "" {
		return next
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Support both X-API-Key header and Authorization header
		providedKey := r.Header.Get("X-API-Key")
		if providedKey == "" {
			// Fallback to Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				slog.Warn("missing authorization header", "path", r.URL.Path, "ip", r.RemoteAddr)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			// Support both "Bearer <key>" and direct key
			providedKey = strings.TrimPrefix(authHeader, "Bearer ")
			providedKey = strings.TrimSpace(providedKey)
		}

		if providedKey != apiKey {
			slog.Warn("invalid API key", "path", r.URL.Path, "ip", r.RemoteAddr)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		slog.Debug("authenticated request", "path", r.URL.Path, "method", r.Method)
		next(w, r)
	}
}
