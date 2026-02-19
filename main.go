package main

import (
	"log/slog"
	"net/http"
	"os"
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
	store, err := NewCortexStore(dbPath)
	if err != nil {
		slog.Error("failed to init cortex store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	mux := http.NewServeMux()

	// Health check (no auth required)
	mux.HandleFunc("/health", store.handleHealth)

	// Neutron-compatible Seeds API
	mux.HandleFunc("/seeds", authMiddleware(methodAllowed(store.handleStoreSeed, http.MethodPost)))
	mux.HandleFunc("/seeds/query", authMiddleware(methodAllowed(store.handleQuerySeed, http.MethodPost)))
	mux.HandleFunc("/seeds/", authMiddleware(methodAllowed(store.handleDeleteSeed, http.MethodDelete)))

	// Cortex API
	mux.HandleFunc("/remember", authMiddleware(methodAllowed(store.handleRemember, http.MethodPost)))
	mux.HandleFunc("/recall", authMiddleware(methodAllowed(store.handleRecall, http.MethodGet)))
	mux.HandleFunc("/entities", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("name") != "" {
				store.handleGetEntity(w, r)
			} else {
				store.handleListEntities(w, r)
			}
		case http.MethodPost, http.MethodPut:
			store.handleSetFact(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/relations", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			store.handleListRelations(w, r)
		case http.MethodPost:
			store.handleAddRelation(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/stats", authMiddleware(methodAllowed(store.handleStats, http.MethodGet)))

	port := os.Getenv("CORTEX_PORT")
	if port == "" {
		port = DefaultPort
	}

	addr := ":" + port
	slog.Info("cortex server starting", "addr", addr, "db", dbPath)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
