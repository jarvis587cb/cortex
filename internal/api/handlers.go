package api

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"cortex/internal/embeddings"
	"cortex/internal/helpers"
	"cortex/internal/models"
	"cortex/internal/store"
)

type Handlers struct {
	store *store.CortexStore
}

func NewHandlers(s *store.CortexStore) *Handlers {
	return &Handlers{store: s}
}

// Health Check

func (h *Handlers) HandleHealth(w http.ResponseWriter, r *http.Request) {
	helpers.WriteJSON(w, http.StatusOK, map[string]string{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Cortex API Handlers

func (h *Handlers) HandleRemember(w http.ResponseWriter, r *http.Request) {
	var req models.RememberRequest
	if err := helpers.ParseJSONBody(r, &req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	if req.Type == "" {
		req.Type = helpers.DefaultMemType
	}
	if req.Importance == 0 {
		req.Importance = helpers.DefaultImportance
	}

	mem := models.Memory{
		Type:       req.Type,
		Content:    req.Content,
		Entity:     req.Entity,
		Tags:       req.Tags,
		Importance: req.Importance,
	}

	if err := h.store.CreateMemory(&mem); err != nil {
		slog.Error("remember insert error", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Generiere Embedding asynchron (nicht-blockierend)
	go func() {
		if err := h.store.GenerateEmbeddingForMemory(&mem); err != nil {
			slog.Warn("failed to generate embedding", "error", err, "memoryId", mem.ID)
		}
	}()

	helpers.WriteJSON(w, http.StatusOK, models.RememberResponse{ID: mem.ID})
}

func (h *Handlers) HandleRecall(w http.ResponseWriter, r *http.Request) {
	query := helpers.GetQueryParam(r, "q")
	memType := helpers.GetQueryParam(r, "type")
	limit := helpers.ParseLimit(helpers.GetQueryParam(r, "limit"), helpers.DefaultLimit, helpers.MaxLimit)

	memories, err := h.store.SearchMemories(query, memType, limit)
	if err != nil {
		slog.Error("recall query error", "error", err, "query", query)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Map metadata for response
	for i := range memories {
		if memories[i].Metadata != "" {
			memories[i].MetadataMap = helpers.UnmarshalMetadata(memories[i].Metadata)
		}
	}

	helpers.WriteJSON(w, http.StatusOK, memories)
}

func (h *Handlers) HandleSetFact(w http.ResponseWriter, r *http.Request) {
	entity := helpers.GetQueryParam(r, "entity")
	if entity == "" {
		http.Error(w, "entity is required (query param)", http.StatusBadRequest)
		return
	}

	var req models.FactRequest
	if err := helpers.ParseJSONBody(r, &req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Key) == "" {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}

	ent, err := h.store.GetEntity(entity)
	data := map[string]any{}
	if err == nil && ent.Data != "" {
		data = helpers.UnmarshalEntityData(ent.Data)
	} else if err != nil && err != gorm.ErrRecordNotFound {
		slog.Error("get entity error", "error", err, "entity", entity)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	data[req.Key] = req.Value
	ent = &models.Entity{
		Name:      entity,
		Data:      helpers.MarshalEntityData(data),
		UpdatedAt: time.Now(),
	}

	if err := h.store.CreateOrUpdateEntity(ent); err != nil {
		slog.Error("set fact error", "error", err, "entity", entity)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) HandleGetEntity(w http.ResponseWriter, r *http.Request) {
	name := helpers.GetQueryParam(r, "name")
	if name == "" {
		http.Error(w, "name is required (query param)", http.StatusBadRequest)
		return
	}

	ent, err := h.store.GetEntity(name)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("get entity error", "error", err, "name", name)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ent.DataMap = helpers.UnmarshalEntityData(ent.Data)
	helpers.WriteJSON(w, http.StatusOK, ent)
}

func (h *Handlers) HandleListEntities(w http.ResponseWriter, r *http.Request) {
	entities, err := h.store.ListEntities()
	if err != nil {
		slog.Error("list entities error", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	for i := range entities {
		entities[i].DataMap = helpers.UnmarshalEntityData(entities[i].Data)
	}

	helpers.WriteJSON(w, http.StatusOK, entities)
}

func (h *Handlers) HandleAddRelation(w http.ResponseWriter, r *http.Request) {
	var req models.RelationRequest
	if err := helpers.ParseJSONBody(r, &req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.From) == "" || strings.TrimSpace(req.To) == "" || strings.TrimSpace(req.Type) == "" {
		http.Error(w, "from, to and type are required", http.StatusBadRequest)
		return
	}

	rel := models.Relation{
		From: req.From,
		To:   req.To,
		Type: req.Type,
	}

	if err := h.store.CreateOrUpdateRelation(&rel); err != nil {
		slog.Error("add relation error", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) HandleListRelations(w http.ResponseWriter, r *http.Request) {
	entity := helpers.GetQueryParam(r, "entity")
	relations, err := h.store.GetRelations(entity)
	if err != nil {
		slog.Error("list relations error", "error", err, "entity", entity)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, relations)
}

func (h *Handlers) HandleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.GetStats()
	if err != nil {
		slog.Error("stats error", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, stats)
}

// Neutron-compatible Seeds API Handlers

func (h *Handlers) HandleStoreSeed(w http.ResponseWriter, r *http.Request) {
	var req models.StoreSeedRequest
	if err := helpers.ParseJSONBody(r, &req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	fields := map[string]string{
		"appId":          req.AppID,
		"externalUserId": req.ExternalUserID,
		"content":        req.Content,
	}
	if field, ok := helpers.ValidateRequired(fields); !ok {
		http.Error(w, "missing required field: "+field, http.StatusBadRequest)
		return
	}

	mem := models.Memory{
		Type:           helpers.DefaultMemType,
		Content:        req.Content,
		AppID:          req.AppID,
		ExternalUserID: req.ExternalUserID,
		Metadata:       helpers.MarshalMetadata(req.Metadata),
		Importance:     helpers.DefaultImportance,
	}

	if err := h.store.CreateMemory(&mem); err != nil {
		slog.Error("store seed error", "error", err, "appId", req.AppID, "userId", req.ExternalUserID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Generiere Embedding asynchron (nicht-blockierend)
	go func() {
		if err := h.store.GenerateEmbeddingForMemory(&mem); err != nil {
			slog.Warn("failed to generate embedding", "error", err, "memoryId", mem.ID)
		}
	}()

	helpers.WriteJSON(w, http.StatusOK, models.StoreSeedResponse{
		ID:      mem.ID,
		Message: "Memory stored successfully",
	})
}

func (h *Handlers) HandleQuerySeed(w http.ResponseWriter, r *http.Request) {
	var req models.QuerySeedRequest
	if err := helpers.ParseJSONBody(r, &req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	fields := map[string]string{
		"appId":          req.AppID,
		"externalUserId": req.ExternalUserID,
		"query":          req.Query,
	}
	if field, ok := helpers.ValidateRequired(fields); !ok {
		http.Error(w, "missing required field: "+field, http.StatusBadRequest)
		return
	}

	limit := req.Limit
	if limit <= 0 || limit > helpers.MaxLimit {
		limit = 5
	}

	// Versuche semantische Suche, fallback zu Textsuche
	memories, err := h.store.SearchMemoriesByTenantSemantic(req.AppID, req.ExternalUserID, req.Query, limit)
	if err != nil {
		// Fallback zu Textsuche
		memories, err = h.store.SearchMemoriesByTenant(req.AppID, req.ExternalUserID, req.Query, limit)
		if err != nil {
			slog.Error("query seed error", "error", err, "appId", req.AppID, "userId", req.ExternalUserID, "query", req.Query)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	// Generiere Query-Embedding für Similarity-Berechnung
	embeddingService := embeddings.GetEmbeddingService()
	queryEmbedding, err := embeddingService.GenerateEmbedding(req.Query, "text/plain")
	if err != nil {
		slog.Warn("failed to generate query embedding, using text similarity", "error", err)
	}

	results := make([]models.QuerySeedResult, 0, len(memories))
	for _, mem := range memories {
		metadata := helpers.UnmarshalMetadata(mem.Metadata)

		// Berechne echte Similarity wenn möglich
		similarity := 0.5 // Default
		if queryEmbedding != nil && mem.Embedding != "" {
			memEmbedding, err := embeddings.DecodeVector(mem.Embedding)
			if err == nil {
				similarity = embeddings.CosineSimilarity(queryEmbedding, memEmbedding)
			}
		} else {
			// Fallback: Text-basierte Similarity
			if strings.Contains(strings.ToLower(mem.Content), strings.ToLower(req.Query)) {
				similarity = 0.8
			}
		}

		results = append(results, models.QuerySeedResult{
			ID:         mem.ID,
			Content:    mem.Content,
			Metadata:   metadata,
			CreatedAt:  mem.CreatedAt,
			Similarity: similarity,
		})
	}

	helpers.WriteJSON(w, http.StatusOK, results)
}

// HandleGenerateEmbeddings generiert Embeddings für alle Memories ohne Embedding
func (h *Handlers) HandleGenerateEmbeddings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	batchSize := helpers.ParseLimit(helpers.GetQueryParam(r, "batchSize"), 10, 100)

	if err := h.store.BatchGenerateEmbeddings(batchSize); err != nil {
		slog.Error("batch generate embeddings error", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Embeddings generation started",
	})
}

func (h *Handlers) HandleDeleteSeed(w http.ResponseWriter, r *http.Request) {
	idStr, err := helpers.ExtractPathID(r.URL.Path, "/seeds/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := helpers.ParseID(idStr)
	if err != nil {
		http.Error(w, "invalid id format", http.StatusBadRequest)
		return
	}

	appID := helpers.GetQueryParam(r, "appId")
	externalUserID := helpers.GetQueryParam(r, "externalUserId")

	fields := map[string]string{
		"appId":          appID,
		"externalUserId": externalUserID,
	}
	if field, ok := helpers.ValidateRequired(fields); !ok {
		http.Error(w, "missing required query parameter: "+field, http.StatusBadRequest)
		return
	}

	mem, err := h.store.GetMemoryByIDAndTenant(id, appID, externalUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Memory not found", http.StatusNotFound)
			return
		}
		slog.Error("delete seed error", "error", err, "id", id, "appId", appID, "userId", externalUserID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.store.DeleteMemory(mem); err != nil {
		slog.Error("delete seed error", "error", err, "id", mem.ID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, models.DeleteSeedResponse{
		Message: "Memory deleted successfully",
		ID:      mem.ID,
	})
}
