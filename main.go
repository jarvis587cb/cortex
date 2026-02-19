package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	dbPath := os.Getenv("CORTEX_DB_PATH")
	store, err := NewCortexStore(dbPath)
	if err != nil {
		log.Fatalf("failed to init cortex store: %v", err)
	}
	defer store.Close()

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", store.handleHealth)

	// Neutron-compatible Seeds API
	mux.HandleFunc("/seeds", methodAllowed(store.handleStoreSeed, http.MethodPost))
	mux.HandleFunc("/seeds/query", methodAllowed(store.handleQuerySeed, http.MethodPost))
	mux.HandleFunc("/seeds/", methodAllowed(store.handleDeleteSeed, http.MethodDelete))

	// Cortex API
	mux.HandleFunc("/remember", methodAllowed(store.handleRemember, http.MethodPost))
	mux.HandleFunc("/recall", methodAllowed(store.handleRecall, http.MethodGet))
	mux.HandleFunc("/entities", func(w http.ResponseWriter, r *http.Request) {
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
	})
	mux.HandleFunc("/relations", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			store.handleListRelations(w, r)
		case http.MethodPost:
			store.handleAddRelation(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/stats", methodAllowed(store.handleStats, http.MethodGet))

	port := os.Getenv("CORTEX_PORT")
	if port == "" {
		port = DefaultPort
	}

	addr := ":" + port
	log.Printf("cortex listening on %s (db: %s)", addr, dbPath)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
