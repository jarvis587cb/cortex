package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// responseRecorder captures status code and size for logging
type responseRecorder struct {
	http.ResponseWriter
	status int
	size   int64
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.size += int64(n)
	return n, err
}

// LoggingMiddleware logs each request to the console (method, path, status, size).
// If the handler panics, the panic is logged and re-raised.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			if err := recover(); err != nil {
				slog.Error("handler panic", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr, "panic", err)
				panic(err)
			}
		}()
		next.ServeHTTP(rec, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"size", rec.size,
			"remote", r.RemoteAddr,
		)
	})
}

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
// If CORTEX_API_KEY is empty, all requests are allowed (local/dev mode - no API key required).
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

// CORSMiddleware sets CORS headers when CORTEX_CORS_ORIGIN is set (e.g. http://localhost:5173 for dashboard dev).
// If unset, the handler is unchanged.
func CORSMiddleware(next http.Handler) http.Handler {
	origin := os.Getenv("CORTEX_CORS_ORIGIN")
	if origin == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
