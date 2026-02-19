package main

import (
	"log/slog"
	"net/http"
	"os"

	"cortex/internal/api"
	"cortex/internal/helpers"
	"cortex/internal/middleware"
	"cortex/internal/store"
)

func main() {
	// Setup structured logging
	logLevel := os.Getenv("CORTEX_LOG_LEVEL")
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	dbPath := os.Getenv("CORTEX_DB_PATH")
	cortexStore, err := store.NewCortexStore(dbPath)
	if err != nil {
		slog.Error("failed to init cortex store", "error", err)
		os.Exit(1)
	}
	defer cortexStore.Close()

	handlers := api.NewHandlers(cortexStore)
	mux := http.NewServeMux()

	// Health check (no auth required)
	mux.HandleFunc("/health", handlers.HandleHealth)

	// Neutron-compatible Seeds API
	mux.HandleFunc("/seeds", middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleStoreSeed, http.MethodPost)))
	mux.HandleFunc("/seeds/query", middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleQuerySeed, http.MethodPost)))
	mux.HandleFunc("/seeds/", middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleDeleteSeed, http.MethodDelete)))

	// Cortex API
	mux.HandleFunc("/remember", middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleRemember, http.MethodPost)))
	mux.HandleFunc("/recall", middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleRecall, http.MethodGet)))
	mux.HandleFunc("/entities", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("name") != "" {
				handlers.HandleGetEntity(w, r)
			} else {
				handlers.HandleListEntities(w, r)
			}
		case http.MethodPost, http.MethodPut:
			handlers.HandleSetFact(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/relations", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.HandleListRelations(w, r)
		case http.MethodPost:
			handlers.HandleAddRelation(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/stats", middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleStats, http.MethodGet)))

	port := os.Getenv("CORTEX_PORT")
	if port == "" {
		port = helpers.DefaultPort
	}

	addr := ":" + port
	slog.Info("cortex server starting", "addr", addr, "db", dbPath)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
