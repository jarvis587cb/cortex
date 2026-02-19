package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"cortex/internal/embeddings"
	"cortex/internal/helpers"
	"cortex/internal/models"
	"cortex/internal/store"
	"cortex/internal/webhooks"
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
	} else if err != nil && !helpers.IsNotFoundError(err) {
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
		if helpers.IsNotFoundError(err) {
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

	// Query-Parameter haben Priorität (Neutron-kompatibel), Fallback zu Body
	appID, externalUserID := helpers.ExtractTenantParams(r, &req)

	fields := map[string]string{
		"appId":          appID,
		"externalUserId": externalUserID,
		"content":        req.Content,
	}
	if field, ok := helpers.ValidateRequired(fields); !ok {
		http.Error(w, "missing required field: "+field, http.StatusBadRequest)
		return
	}

	mem := models.Memory{
		Type:           helpers.DefaultMemType,
		Content:        req.Content,
		AppID:          appID,
		ExternalUserID: externalUserID,
		BundleID:       req.BundleID,
		Metadata:       helpers.MarshalMetadata(req.Metadata),
		Importance:     helpers.DefaultImportance,
	}

	if err := h.store.CreateMemory(&mem); err != nil {
		slog.Error("store seed error", "error", err, "appId", appID, "userId", externalUserID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Generiere Embedding asynchron (nicht-blockierend)
	go func() {
		if err := h.store.GenerateEmbeddingForMemory(&mem); err != nil {
			slog.Warn("failed to generate embedding", "error", err, "memoryId", mem.ID)
		}
	}()

	// Trigger webhook asynchron
	go h.triggerWebhook(webhooks.EventMemoryCreated, map[string]interface{}{
		"id":             mem.ID,
		"app_id":         appID,
		"external_user_id": externalUserID,
		"content":        mem.Content,
		"bundle_id":      mem.BundleID,
		"created_at":     mem.CreatedAt,
	})

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

	// Query-Parameter haben Priorität (Neutron-kompatibel), Fallback zu Body
	appID, externalUserID := helpers.ExtractTenantParams(r, &req)

	fields := map[string]string{
		"appId":          appID,
		"externalUserId": externalUserID,
		"query":          req.Query,
	}
	if field, ok := helpers.ValidateRequired(fields); !ok {
		http.Error(w, "missing required field: "+field, http.StatusBadRequest)
		return
	}

	limit := req.Limit
	if limit <= 0 || limit > helpers.MaxLimit {
		limit = helpers.DefaultQueryLimit
	}

	// Versuche semantische Suche, fallback zu Textsuche
	memories, err := h.store.SearchMemoriesByTenantSemanticAndBundle(appID, externalUserID, req.Query, req.BundleID, limit)
	if err != nil {
		// Fallback zu Textsuche
		memories, err = h.store.SearchMemoriesByTenantAndBundle(appID, externalUserID, req.Query, req.BundleID, limit)
		if err != nil {
			slog.Error("query seed error", "error", err, "appId", appID, "userId", externalUserID, "query", req.Query)
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

	appID, externalUserID := helpers.ExtractTenantParams(r, nil)

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
		if helpers.IsNotFoundError(err) {
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

	// Trigger webhook asynchron
	go h.triggerWebhook(webhooks.EventMemoryDeleted, map[string]interface{}{
		"id":             mem.ID,
		"app_id":         appID,
		"external_user_id": externalUserID,
	})

	helpers.WriteJSON(w, http.StatusOK, models.DeleteSeedResponse{
		Message: "Memory deleted successfully",
		ID:      mem.ID,
	})
}

// Bundle API Handlers

func (h *Handlers) HandleCreateBundle(w http.ResponseWriter, r *http.Request) {
	var req models.CreateBundleRequest
	if err := helpers.ParseJSONBody(r, &req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	// Query-Parameter haben Priorität (Neutron-kompatibel), Fallback zu Body
	appID, externalUserID := helpers.ExtractTenantParams(r, &req)

	fields := map[string]string{
		"appId":          appID,
		"externalUserId": externalUserID,
		"name":           req.Name,
	}
	if field, ok := helpers.ValidateRequired(fields); !ok {
		http.Error(w, "missing required field: "+field, http.StatusBadRequest)
		return
	}

	bundle := models.Bundle{
		Name:           req.Name,
		AppID:          appID,
		ExternalUserID: externalUserID,
	}

	if err := h.store.CreateBundle(&bundle); err != nil {
		slog.Error("create bundle error", "error", err, "appId", appID, "userId", externalUserID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Trigger webhook asynchron
	go h.triggerWebhook(webhooks.EventBundleCreated, map[string]interface{}{
		"id":               bundle.ID,
		"name":             bundle.Name,
		"app_id":           bundle.AppID,
		"external_user_id": bundle.ExternalUserID,
		"created_at":       bundle.CreatedAt,
	})

	helpers.WriteJSON(w, http.StatusOK, models.BundleResponse{
		ID:             bundle.ID,
		Name:           bundle.Name,
		AppID:          bundle.AppID,
		ExternalUserID: bundle.ExternalUserID,
		CreatedAt:      bundle.CreatedAt,
	})
}

func (h *Handlers) HandleListBundles(w http.ResponseWriter, r *http.Request) {
	appID, externalUserID := helpers.ExtractTenantParams(r, nil)

	fields := map[string]string{
		"appId":          appID,
		"externalUserId": externalUserID,
	}
	if field, ok := helpers.ValidateRequired(fields); !ok {
		http.Error(w, "missing required query parameter: "+field, http.StatusBadRequest)
		return
	}

	bundles, err := h.store.ListBundles(appID, externalUserID)
	if err != nil {
		slog.Error("list bundles error", "error", err, "appId", appID, "userId", externalUserID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	responses := make([]models.BundleResponse, len(bundles))
	for i, bundle := range bundles {
		responses[i] = models.BundleResponse{
			ID:             bundle.ID,
			Name:           bundle.Name,
			AppID:          bundle.AppID,
			ExternalUserID: bundle.ExternalUserID,
			CreatedAt:      bundle.CreatedAt,
		}
	}

	helpers.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handlers) HandleGetBundle(w http.ResponseWriter, r *http.Request) {
	idStr, err := helpers.ExtractPathID(r.URL.Path, "/bundles/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := helpers.ParseID(idStr)
	if err != nil {
		http.Error(w, "invalid id format", http.StatusBadRequest)
		return
	}

	appID, externalUserID := helpers.ExtractTenantParams(r, nil)

	fields := map[string]string{
		"appId":          appID,
		"externalUserId": externalUserID,
	}
	if field, ok := helpers.ValidateRequired(fields); !ok {
		http.Error(w, "missing required query parameter: "+field, http.StatusBadRequest)
		return
	}

	bundle, err := h.store.GetBundle(id, appID, externalUserID)
	if err != nil {
		if helpers.IsNotFoundError(err) {
			http.Error(w, "Bundle not found", http.StatusNotFound)
			return
		}
		slog.Error("get bundle error", "error", err, "id", id, "appId", appID, "userId", externalUserID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, models.BundleResponse{
		ID:             bundle.ID,
		Name:           bundle.Name,
		AppID:          bundle.AppID,
		ExternalUserID: bundle.ExternalUserID,
		CreatedAt:      bundle.CreatedAt,
	})
}

func (h *Handlers) HandleDeleteBundle(w http.ResponseWriter, r *http.Request) {
	idStr, err := helpers.ExtractPathID(r.URL.Path, "/bundles/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := helpers.ParseID(idStr)
	if err != nil {
		http.Error(w, "invalid id format", http.StatusBadRequest)
		return
	}

	appID, externalUserID := helpers.ExtractTenantParams(r, nil)

	fields := map[string]string{
		"appId":          appID,
		"externalUserId": externalUserID,
	}
	if field, ok := helpers.ValidateRequired(fields); !ok {
		http.Error(w, "missing required query parameter: "+field, http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteBundle(id, appID, externalUserID); err != nil {
		if helpers.IsNotFoundError(err) {
			http.Error(w, "Bundle not found", http.StatusNotFound)
			return
		}
		slog.Error("delete bundle error", "error", err, "id", id, "appId", appID, "userId", externalUserID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Trigger webhook asynchron
	go h.triggerWebhook(webhooks.EventBundleDeleted, map[string]interface{}{
		"id":               id,
		"app_id":           appID,
		"external_user_id": externalUserID,
	})

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Bundle deleted successfully",
		"id":      id,
	})
}

// Webhook API Handlers

func (h *Handlers) HandleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	var req models.CreateWebhookRequest
	if err := helpers.ParseJSONBody(r, &req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	if len(req.Events) == 0 {
		http.Error(w, "events are required", http.StatusBadRequest)
		return
	}

	webhook := models.Webhook{
		URL:    req.URL,
		Events: strings.Join(req.Events, ","),
		Secret: req.Secret,
		AppID:  req.AppID,
		Active: true,
	}

	if err := h.store.CreateWebhook(&webhook); err != nil {
		slog.Error("create webhook error", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, models.WebhookResponse{
		ID:        webhook.ID,
		URL:       webhook.URL,
		Events:    req.Events,
		AppID:     webhook.AppID,
		Active:    webhook.Active,
		CreatedAt: webhook.CreatedAt,
		UpdatedAt: webhook.UpdatedAt,
	})
}

func (h *Handlers) HandleListWebhooks(w http.ResponseWriter, r *http.Request) {
	appID := helpers.GetQueryParam(r, "appId")

	webhookList, err := h.store.ListWebhooks(appID)
	if err != nil {
		slog.Error("list webhooks error", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	responses := make([]models.WebhookResponse, len(webhookList))
	for i, wh := range webhookList {
		events := strings.Split(wh.Events, ",")
		responses[i] = models.WebhookResponse{
			ID:        wh.ID,
			URL:       wh.URL,
			Events:    events,
			AppID:     wh.AppID,
			Active:    wh.Active,
			CreatedAt: wh.CreatedAt,
			UpdatedAt: wh.UpdatedAt,
		}
	}

	helpers.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handlers) HandleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	idStr, err := helpers.ExtractPathID(r.URL.Path, "/webhooks/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := helpers.ParseID(idStr)
	if err != nil {
		http.Error(w, "invalid id format", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteWebhook(id); err != nil {
		if helpers.IsNotFoundError(err) {
			http.Error(w, "Webhook not found", http.StatusNotFound)
			return
		}
		slog.Error("delete webhook error", "error", err, "id", id)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Webhook deleted successfully",
		"id":      id,
	})
}

// Export/Import API Handlers

func (h *Handlers) HandleExport(w http.ResponseWriter, r *http.Request) {
	appID, externalUserID := helpers.ExtractTenantParams(r, nil)

	fields := map[string]string{
		"appId":          appID,
		"externalUserId": externalUserID,
	}
	if field, ok := helpers.ValidateRequired(fields); !ok {
		http.Error(w, "missing required query parameter: "+field, http.StatusBadRequest)
		return
	}

	exportData, err := h.store.ExportAll(appID, externalUserID)
	if err != nil {
		slog.Error("export error", "error", err, "appId", appID, "userId", externalUserID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"cortex-export-%s-%s-%s.json\"", appID, externalUserID, time.Now().Format("20060102-150405")))
	helpers.WriteJSON(w, http.StatusOK, exportData)
}

func (h *Handlers) HandleImport(w http.ResponseWriter, r *http.Request) {
	appID, externalUserID := helpers.ExtractTenantParams(r, nil)

	fields := map[string]string{
		"appId":          appID,
		"externalUserId": externalUserID,
	}
	if field, ok := helpers.ValidateRequired(fields); !ok {
		http.Error(w, "missing required query parameter: "+field, http.StatusBadRequest)
		return
	}

	var exportData store.ExportData
	if err := helpers.ParseJSONBody(r, &exportData); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	overwrite := helpers.GetQueryParam(r, "overwrite") == "true"

	if err := h.store.ImportData(&exportData, overwrite); err != nil {
		slog.Error("import error", "error", err, "appId", appID, "userId", externalUserID)
		http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Import completed successfully",
		"memories":  len(exportData.Memories),
		"bundles":   len(exportData.Bundles),
		"webhooks":  len(exportData.Webhooks),
		"overwrite": overwrite,
	})
}

// Backup/Restore API Handlers

func (h *Handlers) HandleBackup(w http.ResponseWriter, r *http.Request) {
	backupPath := helpers.GetQueryParam(r, "path")
	if backupPath == "" {
		// Default backup path
		backupPath = fmt.Sprintf("cortex-backup-%s.db", time.Now().Format("20060102-150405"))
	}

	if err := h.store.BackupDatabase(backupPath); err != nil {
		slog.Error("backup error", "error", err, "path", backupPath)
		http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Backup created successfully",
		"path":    backupPath,
	})
}

func (h *Handlers) HandleRestore(w http.ResponseWriter, r *http.Request) {
	backupPath := helpers.GetQueryParam(r, "path")
	if backupPath == "" {
		http.Error(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	// Get current database path
	currentPath, err := h.store.GetDatabasePath()
	if err != nil {
		slog.Error("get database path error", "error", err)
		http.Error(w, "internal error: failed to get database path", http.StatusInternalServerError)
		return
	}

	// Check if backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		http.Error(w, "backup file not found", http.StatusNotFound)
		return
	}

	// Note: Restore requires server restart. We'll just copy the file
	// and inform the user that a restart is needed.
	if err := h.store.CopyFile(backupPath, currentPath); err != nil {
		slog.Error("restore error", "error", err, "backupPath", backupPath, "currentPath", currentPath)
		http.Error(w, "internal error: failed to restore database: "+err.Error(), http.StatusInternalServerError)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message":     "Restore completed successfully. Server restart required to use restored database.",
		"backup_path": backupPath,
		"restored_to": currentPath,
		"warning":     "Server must be restarted for changes to take effect",
	})
}

// Analytics API Handlers

func (h *Handlers) HandleAnalytics(w http.ResponseWriter, r *http.Request) {
	appID := helpers.GetQueryParam(r, "appId")
	externalUserID := helpers.GetQueryParam(r, "externalUserId")
	daysStr := helpers.GetQueryParam(r, "days")

	days := helpers.DefaultAnalyticsDays
	if daysStr != "" {
		if d := helpers.ParseLimit(daysStr, 1, 365); d > 0 {
			days = d
		}
	}

	var analytics *store.AnalyticsData
	var err error

	if appID != "" && externalUserID != "" {
		// Tenant-specific analytics
		analytics, err = h.store.GetAnalytics(appID, externalUserID, days)
	} else {
		// Global analytics (requires admin or can be restricted)
		analytics, err = h.store.GetGlobalAnalytics(days)
	}

	if err != nil {
		slog.Error("analytics error", "error", err, "appId", appID, "userId", externalUserID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, analytics)
}

// triggerWebhook triggers webhooks for a given event
func (h *Handlers) triggerWebhook(event webhooks.EventType, data map[string]interface{}) {
	// Get app_id from data if available
	appID := ""
	if id, ok := data["app_id"].(string); ok {
		appID = id
	}

	// Get active webhooks
	webhookList, err := h.store.ListWebhooks(appID)
	if err != nil {
		slog.Warn("failed to list webhooks", "error", err)
		return
	}

	// Convert to webhook configs
	configs := make([]webhooks.WebhookConfig, 0, len(webhookList))
	for _, wh := range webhookList {
		if !wh.Active {
			continue
		}

		// Parse events
		events := strings.Split(wh.Events, ",")
		eventTypes := make([]webhooks.EventType, 0, len(events))
		for _, e := range events {
			eventTypes = append(eventTypes, webhooks.EventType(strings.TrimSpace(e)))
		}

		configs = append(configs, webhooks.WebhookConfig{
			URL:    wh.URL,
			Secret: wh.Secret,
			Events: eventTypes,
		})
	}

	// Deliver webhooks asynchronously
	webhooks.DeliverWebhooksAsync(configs, event, data)
}
