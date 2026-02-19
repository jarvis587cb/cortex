package middleware

import (
	"log/slog"
	"net/http"
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

// AuthMiddleware is a pass-through (no authentication; API key was removed from the project)
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return next
}
