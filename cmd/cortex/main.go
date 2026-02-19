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

	// Health check (no auth required, no rate limit)
	mux.HandleFunc("/health", handlers.HandleHealth)

	// Neutron-compatible Seeds API (with rate limiting)
	mux.HandleFunc("/seeds", middleware.RateLimitMiddleware(middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleStoreSeed, http.MethodPost))))
	mux.HandleFunc("/seeds/query", middleware.RateLimitMiddleware(middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleQuerySeed, http.MethodPost))))
	mux.HandleFunc("/seeds/generate-embeddings", middleware.RateLimitMiddleware(middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleGenerateEmbeddings, http.MethodPost))))
	mux.HandleFunc("/seeds/", middleware.RateLimitMiddleware(middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleDeleteSeed, http.MethodDelete))))

	// Bundles API (with rate limiting)
	// Register /bundles/ first to avoid routing conflicts
	mux.HandleFunc("/bundles/", middleware.RateLimitMiddleware(middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.HandleGetBundle(w, r)
		case http.MethodDelete:
			handlers.HandleDeleteBundle(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	// Register /bundles after /bundles/ to ensure exact match
	mux.HandleFunc("/bundles", middleware.RateLimitMiddleware(middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.HandleCreateBundle(w, r)
		case http.MethodGet:
			handlers.HandleListBundles(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))

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
	mux.HandleFunc("/stats", middleware.RateLimitMiddleware(middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleStats, http.MethodGet))))

	// Webhooks API (with rate limiting)
	mux.HandleFunc("/webhooks", middleware.RateLimitMiddleware(middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.HandleCreateWebhook(w, r)
		case http.MethodGet:
			handlers.HandleListWebhooks(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	mux.HandleFunc("/webhooks/", middleware.RateLimitMiddleware(middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleDeleteWebhook, http.MethodDelete))))

	// Export/Import API (with rate limiting)
	mux.HandleFunc("/export", middleware.RateLimitMiddleware(middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleExport, http.MethodGet))))
	mux.HandleFunc("/import", middleware.RateLimitMiddleware(middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleImport, http.MethodPost))))

	// Backup/Restore API (with rate limiting)
	mux.HandleFunc("/backup", middleware.RateLimitMiddleware(middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleBackup, http.MethodPost))))
	mux.HandleFunc("/restore", middleware.RateLimitMiddleware(middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleRestore, http.MethodPost))))

	// Analytics API (with rate limiting)
	mux.HandleFunc("/analytics", middleware.RateLimitMiddleware(middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleAnalytics, http.MethodGet))))

	// Agent Contexts API (Neutron-compatible, with rate limiting)
	// Exact /agent-contexts first so GET/POST without trailing slash match
	mux.HandleFunc("/agent-contexts", middleware.RateLimitMiddleware(middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.HandleCreateAgentContext(w, r)
		case http.MethodGet:
			handlers.HandleListAgentContexts(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	mux.HandleFunc("/agent-contexts/", middleware.RateLimitMiddleware(middleware.AuthMiddleware(middleware.MethodAllowed(handlers.HandleGetAgentContext, http.MethodGet))))

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
