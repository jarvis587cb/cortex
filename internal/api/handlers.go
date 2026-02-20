package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

// generateEmbeddingAsync generates embedding for a memory asynchronously
func (h *Handlers) generateEmbeddingAsync(mem *models.Memory) {
	go func() {
		if err := h.store.GenerateEmbeddingForMemory(mem); err != nil {
			slog.Warn("failed to generate embedding", "error", err, "memoryId", mem.ID)
		}
	}()
}

// mapMetadataToMemories maps metadata JSON to MetadataMap for all memories
func (h *Handlers) mapMetadataToMemories(memories []models.Memory) {
	for i := range memories {
		if memories[i].Metadata != "" {
			memories[i].MetadataMap = helpers.UnmarshalMetadata(memories[i].Metadata)
		}
	}
}

// mapEntityDataToEntities maps entity data JSON to DataMap for all entities
func (h *Handlers) mapEntityDataToEntities(entities []models.Entity) {
	for i := range entities {
		entities[i].DataMap = helpers.UnmarshalEntityData(entities[i].Data)
	}
}

// buildMemoryWebhookPayload creates a webhook payload for memory events
func (h *Handlers) buildMemoryWebhookPayload(mem *models.Memory, appID, externalUserID string, eventType webhooks.EventType) map[string]interface{} {
	payload := map[string]interface{}{
		"id":               mem.ID,
		"app_id":           appID,
		"external_user_id": externalUserID,
	}

	if eventType == webhooks.EventMemoryCreated {
		payload["content"] = mem.Content
		payload["bundle_id"] = mem.BundleID
		payload["created_at"] = mem.CreatedAt
	}

	return payload
}

// buildBundleWebhookPayload creates a webhook payload for bundle events
func (h *Handlers) buildBundleWebhookPayload(bundle *models.Bundle, appID, externalUserID string, eventType webhooks.EventType) map[string]interface{} {
	payload := map[string]interface{}{
		"id":               bundle.ID,
		"app_id":           appID,
		"external_user_id": externalUserID,
	}

	if eventType == webhooks.EventBundleCreated {
		payload["name"] = bundle.Name
		payload["created_at"] = bundle.CreatedAt
	}

	return payload
}

// handleStoreOperationWithNotFound handles store operations with NotFound error checking
// Returns true if error was handled (not found or internal error), false if no error
func (h *Handlers) handleStoreOperationWithNotFound(w http.ResponseWriter, err error, resourceName, operationName string, contextArgs ...any) bool {
	if err == nil {
		return false
	}
	if helpers.HandleNotFoundError(w, err, resourceName) {
		return true
	}
	args := append([]any{"error", err, "operation", operationName}, contextArgs...)
	helpers.HandleInternalErrorSlog(w, operationName+" error", args...)
	return true
}

// mapToResponses maps a slice of items to responses using a mapper function
func mapToResponses[T any, R any](items []T, mapper func(T) R) []R {
	responses := make([]R, len(items))
	for i := range items {
		responses[i] = mapper(items[i])
	}
	return responses
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
	if !helpers.ParseJSONBodyOrError(w, r, &req) {
		return
	}

	if !helpers.ValidateNotEmpty(req.Content, "content") {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	if req.Type == "" {
		req.Type = helpers.DefaultMemType
	}
	if req.Importance == 0 {
		req.Importance = helpers.DefaultImportance
	}

	mem := models.NewMemoryFromRememberRequest(&req)

	if err := h.store.CreateMemory(mem); err != nil {
		helpers.HandleInternalErrorSlog(w, "remember insert error", "error", err)
		return
	}

	// Generiere Embedding asynchron (nicht-blockierend)
	h.generateEmbeddingAsync(mem)

	helpers.WriteJSON(w, http.StatusOK, models.RememberResponse{ID: mem.ID})
}

func (h *Handlers) HandleRecall(w http.ResponseWriter, r *http.Request) {
	query := helpers.GetQueryParam(r, "q")
	memType := helpers.GetQueryParam(r, "type")
	limit := helpers.ParseLimit(helpers.GetQueryParam(r, "limit"), helpers.DefaultLimit, helpers.MaxLimit)

	memories, err := h.store.SearchMemories(query, memType, limit)
	if err != nil {
		helpers.HandleInternalErrorSlog(w, "recall query error", "error", err, "query", query)
		return
	}

	// Map metadata for response
	h.mapMetadataToMemories(memories)

	helpers.WriteJSON(w, http.StatusOK, memories)
}

func (h *Handlers) HandleSetFact(w http.ResponseWriter, r *http.Request) {
	entity := helpers.GetQueryParam(r, "entity")
	if entity == "" {
		http.Error(w, "entity is required (query param)", http.StatusBadRequest)
		return
	}

	var req models.FactRequest
	if !helpers.ParseJSONBodyOrError(w, r, &req) {
		return
	}

	if !helpers.ValidateNotEmpty(req.Key, "key") {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}

	// Use a transaction to prevent race conditions when updating entity facts
	// SQLite handles transactions atomically, preventing concurrent modification issues
	err := h.store.GetDB().Transaction(func(tx *gorm.DB) error {
		var ent models.Entity
		// Read existing entity (if any)
		result := tx.Where("name = ?", entity).First(&ent)
		
		data := map[string]any{}
		if result.Error == nil {
			// Entity exists, unmarshal existing data
			if ent.Data != "" {
				data = helpers.UnmarshalEntityData(ent.Data)
			}
		} else if result.Error != gorm.ErrRecordNotFound {
			// Unexpected error
			return result.Error
		}
		
		// Add or update the fact
		data[req.Key] = req.Value
		
		// Update entity with new data
		ent.Name = entity
		ent.Data = helpers.MarshalEntityData(data)
		ent.UpdatedAt = time.Now()
		
		// Use ON CONFLICT to handle concurrent inserts atomically
		// If entity exists, update; if not, insert
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"data", "updated_at"}),
		}).Create(&ent).Error
	})
	
	if err != nil {
		helpers.HandleInternalErrorSlog(w, "set fact error", "error", err, "entity", entity)
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
		if helpers.HandleNotFoundError(w, err, "Entity") {
			return
		}
		helpers.HandleInternalErrorSlog(w, "get entity error", "error", err, "name", name)
		return
	}

	ent.DataMap = helpers.UnmarshalEntityData(ent.Data)
	helpers.WriteJSON(w, http.StatusOK, ent)
}

func (h *Handlers) HandleListEntities(w http.ResponseWriter, r *http.Request) {
	entities, err := h.store.ListEntities()
	if err != nil {
		helpers.HandleInternalErrorSlog(w, "list entities error", "error", err)
		return
	}

	h.mapEntityDataToEntities(entities)

	helpers.WriteJSON(w, http.StatusOK, entities)
}

func (h *Handlers) HandleAddRelation(w http.ResponseWriter, r *http.Request) {
	var req models.RelationRequest
	if !helpers.ParseJSONBodyOrError(w, r, &req) {
		return
	}

	if !helpers.ValidateNotEmpty(req.From, "from") || !helpers.ValidateNotEmpty(req.To, "to") || !helpers.ValidateNotEmpty(req.Type, "type") {
		http.Error(w, "from, to and type are required", http.StatusBadRequest)
		return
	}

	rel := models.Relation{
		From: req.From,
		To:   req.To,
		Type: req.Type,
	}

	if err := h.store.CreateOrUpdateRelation(&rel); err != nil {
		helpers.HandleInternalErrorSlog(w, "add relation error", "error", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) HandleListRelations(w http.ResponseWriter, r *http.Request) {
	entity := helpers.GetQueryParam(r, "entity")
	relations, err := h.store.GetRelations(entity)
	if err != nil {
		helpers.HandleInternalErrorSlog(w, "list relations error", "error", err, "entity", entity)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, relations)
}

func (h *Handlers) HandleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.GetStats()
	if err != nil {
		helpers.HandleInternalErrorSlog(w, "stats error", "error", err)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, stats)
}

// Neutron-compatible Seeds API Handlers

func (h *Handlers) HandleStoreSeed(w http.ResponseWriter, r *http.Request) {
	var req models.StoreSeedRequest
	if !helpers.ParseJSONBodyOrError(w, r, &req) {
		return
	}

	// Query-Parameter haben Priorität (Neutron-kompatibel), Fallback zu Body
	appID, externalUserID, ok := helpers.ValidateTenantParamsWithFields(w, r, &req, map[string]string{"content": req.Content}, false)
	if !ok {
		return
	}

	mem := models.NewMemoryFromStoreSeedRequest(&req, appID, externalUserID)

	if err := h.store.CreateMemory(mem); err != nil {
		helpers.HandleInternalErrorSlog(w, "store seed error", "error", err, "appId", appID, "userId", externalUserID)
		return
	}

	// Embedding synchron erzeugen, damit semantische Suche sofort Treffer liefert
	if err := h.store.GenerateEmbeddingForMemory(mem); err != nil {
		slog.Warn("embedding on store failed, memory still saved", "error", err, "memoryId", mem.ID)
	}

	// Trigger webhook asynchron
	go h.triggerWebhook(webhooks.EventMemoryCreated, h.buildMemoryWebhookPayload(mem, appID, externalUserID, webhooks.EventMemoryCreated))

	helpers.WriteJSON(w, http.StatusOK, helpers.NewSuccessResponse(mem.ID, "Memory stored successfully"))
}

func (h *Handlers) HandleQuerySeed(w http.ResponseWriter, r *http.Request) {
	var req models.QuerySeedRequest
	if !helpers.ParseJSONBodyOrError(w, r, &req) {
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

	// Optional: limit search to specific seed IDs (Neutron-compatible)
	seedIDs := req.SeedIDs
	if seedIDs == nil {
		seedIDs = []int64{}
	}

	// Versuche semantische Suche, fallback zu Textsuche (bei Fehler oder 0 Treffern)
	memories, err := h.store.SearchMemoriesByTenantSemanticAndBundle(appID, externalUserID, req.Query, req.BundleID, limit, seedIDs)
	if err != nil || len(memories) == 0 {
		// Fallback zu Textsuche (z. B. wenn noch keine Embeddings vorhanden oder semantisch nichts gefunden)
		textMemories, textErr := h.store.SearchMemoriesByTenantAndBundle(appID, externalUserID, req.Query, req.BundleID, limit, seedIDs)
		if textErr != nil {
			if err != nil {
				helpers.HandleInternalErrorSlog(w, "query seed error", "error", err, "appId", appID, "userId", externalUserID, "query", req.Query)
				return
			}
			// err war nil (leere semantische Liste), nur Textsuche fehlgeschlagen → nutze leere Liste
		} else {
			memories = textMemories
		}
	}

	// Generiere Query-Embedding für Similarity-Berechnung
	embeddingService := embeddings.GetEmbeddingService()
	queryEmbedding, err := embeddingService.GenerateEmbedding(req.Query, "text/plain")
	if err != nil {
		slog.Warn("failed to generate query embedding, using text similarity", "error", err)
	}

	// Threshold 0-1: only return results with similarity >= threshold (Neutron-compatible; 0 = no filter)
	threshold := req.Threshold
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 1 {
		threshold = 1
	}

	results := make([]models.QuerySeedResult, 0, len(memories))
	for _, mem := range memories {
		metadata := helpers.UnmarshalMetadata(mem.Metadata)

		// Berechne echte Similarity wenn möglich
		similarity := helpers.DefaultSimilarity
		if queryEmbedding != nil && mem.Embedding != "" {
			memEmbedding, err := embeddings.DecodeVector(mem.Embedding)
			if err == nil {
				similarity = embeddings.CosineSimilarity(queryEmbedding, memEmbedding)
			}
		} else {
			// Fallback: Text-basierte Similarity
			if strings.Contains(strings.ToLower(mem.Content), strings.ToLower(req.Query)) {
				similarity = helpers.TextMatchSimilarity
			}
		}

		if threshold > 0 && similarity < threshold {
			continue
		}

		results = append(results, models.QuerySeedResult{
			ID:         mem.ID,
			Content:    mem.Content,
			Metadata:   metadata,
			CreatedAt:  mem.CreatedAt,
			Similarity: similarity,
		})
	}

	// Wenn nach Threshold 0 Treffer: Textsuche ergänzen (z. B. "oat milk" findet "oat milk lattes")
	if len(results) == 0 && req.Query != "" {
		textMemories, _ := h.store.SearchMemoriesByTenantAndBundle(appID, externalUserID, req.Query, req.BundleID, limit, seedIDs)
		for _, mem := range textMemories {
			metadata := helpers.UnmarshalMetadata(mem.Metadata)
			sim := helpers.DefaultSimilarity
			if strings.Contains(strings.ToLower(mem.Content), strings.ToLower(req.Query)) {
				sim = helpers.TextMatchSimilarity
			}
			if threshold > 0 && sim < threshold {
				continue
			}
			results = append(results, models.QuerySeedResult{
				ID:         mem.ID,
				Content:    mem.Content,
				Metadata:   metadata,
				CreatedAt:  mem.CreatedAt,
				Similarity: sim,
			})
		}
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
		helpers.HandleInternalErrorSlog(w, "batch generate embeddings error", "error", err)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Embeddings generation started",
	})
}

func (h *Handlers) HandleDeleteSeed(w http.ResponseWriter, r *http.Request) {
	id, ok := helpers.ExtractAndParseID(w, r.URL.Path, "/seeds/")
	if !ok {
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
	if h.handleStoreOperationWithNotFound(w, err, "Memory", "delete seed", "id", id, "appId", appID, "userId", externalUserID) {
		return
	}

	if err := h.store.DeleteMemory(mem); err != nil {
		helpers.HandleInternalErrorSlog(w, "delete seed error", "error", err, "id", mem.ID)
		return
	}

	// Trigger webhook asynchron
	go h.triggerWebhook(webhooks.EventMemoryDeleted, h.buildMemoryWebhookPayload(mem, appID, externalUserID, webhooks.EventMemoryDeleted))

	helpers.WriteJSON(w, http.StatusOK, helpers.NewSuccessResponse(mem.ID, "Memory deleted successfully"))
}

// Bundle API Handlers

func (h *Handlers) HandleCreateBundle(w http.ResponseWriter, r *http.Request) {
	var req models.CreateBundleRequest
	if !helpers.ParseJSONBodyOrError(w, r, &req) {
		return
	}

	// Query-Parameter haben Priorität (Neutron-kompatibel), Fallback zu Body
	appID, externalUserID, ok := helpers.ValidateTenantParamsWithFields(w, r, &req, map[string]string{"name": req.Name}, false)
	if !ok {
		return
	}

	bundle := models.Bundle{
		Name:           req.Name,
		AppID:          appID,
		ExternalUserID: externalUserID,
	}

	if err := h.store.CreateBundle(&bundle); err != nil {
		helpers.HandleInternalErrorSlog(w, "create bundle error", "error", err, "appId", appID, "userId", externalUserID)
		return
	}

	// Trigger webhook asynchron
	go h.triggerWebhook(webhooks.EventBundleCreated, h.buildBundleWebhookPayload(&bundle, bundle.AppID, bundle.ExternalUserID, webhooks.EventBundleCreated))

	helpers.WriteJSON(w, http.StatusOK, bundle.ToBundleResponse())
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
		helpers.HandleInternalErrorSlog(w, "list bundles error", "error", err, "appId", appID, "userId", externalUserID)
		return
	}

	responses := mapToResponses(bundles, func(b models.Bundle) models.BundleResponse {
		return b.ToBundleResponse()
	})
	helpers.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handlers) HandleGetBundle(w http.ResponseWriter, r *http.Request) {
	id, ok := helpers.ExtractAndParseID(w, r.URL.Path, "/bundles/")
	if !ok {
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
	if h.handleStoreOperationWithNotFound(w, err, "Bundle", "get bundle", "id", id, "appId", appID, "userId", externalUserID) {
		return
	}

	helpers.WriteJSON(w, http.StatusOK, bundle.ToBundleResponse())
}

func (h *Handlers) HandleDeleteBundle(w http.ResponseWriter, r *http.Request) {
	id, ok := helpers.ExtractAndParseID(w, r.URL.Path, "/bundles/")
	if !ok {
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

	if h.handleStoreOperationWithNotFound(w, h.store.DeleteBundle(id, appID, externalUserID), "Bundle", "delete bundle", "id", id, "appId", appID, "userId", externalUserID) {
		return
	}

	// Trigger webhook asynchron - create minimal bundle for payload
	bundle := &models.Bundle{ID: id, AppID: appID, ExternalUserID: externalUserID}
	go h.triggerWebhook(webhooks.EventBundleDeleted, h.buildBundleWebhookPayload(bundle, appID, externalUserID, webhooks.EventBundleDeleted))

	helpers.WriteJSON(w, http.StatusOK, helpers.NewSuccessResponse(id, "Bundle deleted successfully"))
}

// Webhook API Handlers

func (h *Handlers) HandleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	var req models.CreateWebhookRequest
	if !helpers.ParseJSONBodyOrError(w, r, &req) {
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
		helpers.HandleInternalErrorSlog(w, "create webhook error", "error", err)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, webhook.ToWebhookResponse(strings.Join(req.Events, ",")))
}

func (h *Handlers) HandleListWebhooks(w http.ResponseWriter, r *http.Request) {
	appID := helpers.GetQueryParam(r, "appId")

	webhookList, err := h.store.ListWebhooks(appID)
	if err != nil {
		helpers.HandleInternalErrorSlog(w, "list webhooks error", "error", err)
		return
	}

	responses := mapToResponses(webhookList, func(w models.Webhook) models.WebhookResponse {
		return w.ToWebhookResponse(w.Events)
	})
	helpers.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handlers) HandleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	id, ok := helpers.ExtractAndParseID(w, r.URL.Path, "/webhooks/")
	if !ok {
		return
	}

	if h.handleStoreOperationWithNotFound(w, h.store.DeleteWebhook(id), "Webhook", "delete webhook", "id", id) {
		return
	}

	helpers.WriteJSON(w, http.StatusOK, helpers.NewSuccessResponse(id, "Webhook deleted successfully"))
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
		helpers.HandleInternalErrorSlog(w, "export error", "error", err, "appId", appID, "userId", externalUserID)
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
	if !helpers.ParseJSONBodyOrError(w, r, &exportData) {
		return
	}

	overwrite := helpers.GetQueryParam(r, "overwrite") == "true"

	if err := h.store.ImportData(&exportData, overwrite); err != nil {
		helpers.HandleInternalErrorSlog(w, "import error", "error", err, "appId", appID, "userId", externalUserID)
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
		helpers.HandleInternalErrorSlog(w, "backup error", "error", err, "path", backupPath)
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
		helpers.HandleInternalErrorSlog(w, "get database path error", "error", err)
		return
	}

	// Check if backup file exists
	if !h.store.FileExists(backupPath) {
		http.Error(w, "backup file not found", http.StatusNotFound)
		return
	}

	// Note: Restore requires server restart. We'll just copy the file
	// and inform the user that a restart is needed.
	if err := h.store.CopyFile(backupPath, currentPath); err != nil {
		helpers.HandleInternalErrorSlog(w, "restore error", "error", err, "backupPath", backupPath, "currentPath", currentPath)
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
		helpers.HandleInternalErrorSlog(w, "analytics error", "error", err, "appId", appID, "userId", externalUserID)
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

// HandleCreateAgentContext creates an agent context (Neutron-compatible)
func (h *Handlers) HandleCreateAgentContext(w http.ResponseWriter, r *http.Request) {
	var req models.CreateAgentContextRequest
	if !helpers.ParseJSONBodyOrError(w, r, &req) {
		return
	}
	appID, externalUserID := helpers.ExtractTenantParams(r, &req.TenantRequest)
	fields := map[string]string{
		"appId": appID, "externalUserId": externalUserID,
		"agentId": req.AgentID, "memoryType": req.MemoryType,
	}
	if field, ok := helpers.ValidateRequired(fields); !ok {
		http.Error(w, "missing required field: "+field, http.StatusBadRequest)
		return
	}
	memType := strings.ToLower(strings.TrimSpace(req.MemoryType))
	allowed := map[string]bool{"episodic": true, "semantic": true, "procedural": true, "working": true}
	if !allowed[memType] {
		http.Error(w, "memoryType must be one of: episodic, semantic, procedural, working", http.StatusBadRequest)
		return
	}
	payloadStr := ""
	if len(req.Payload) > 0 {
		b, _ := json.Marshal(req.Payload)
		payloadStr = string(b)
	}
	tagsStr := strings.Join(req.Tags, ",")
	ctx := &models.AgentContext{
		AppID:          appID,
		ExternalUserID: externalUserID,
		AgentID:        req.AgentID,
		MemoryType:     memType,
		Payload:        payloadStr,
		Tags:           tagsStr,
	}
	if err := h.store.CreateAgentContext(ctx); err != nil {
		helpers.HandleInternalErrorSlog(w, "create agent context error", "error", err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, map[string]any{
		"id":         ctx.ID,
		"message":    "Agent context created",
		"created_at": ctx.CreatedAt,
	})
}

// HandleListAgentContexts lists agent contexts (Neutron-compatible)
func (h *Handlers) HandleListAgentContexts(w http.ResponseWriter, r *http.Request) {
	appID := helpers.GetQueryParam(r, "appId")
	externalUserID := helpers.GetQueryParam(r, "externalUserId")
	if appID == "" || externalUserID == "" {
		http.Error(w, "missing required query parameter: appId and externalUserId", http.StatusBadRequest)
		return
	}
	agentID := helpers.GetQueryParam(r, "agentId")
	memoryType := helpers.GetQueryParam(r, "memoryType")
	tags := helpers.GetQueryParam(r, "tags")
	list, err := h.store.ListAgentContexts(appID, externalUserID, agentID, memoryType, tags)
	if err != nil {
		helpers.HandleInternalErrorSlog(w, "list agent contexts error", "error", err)
		return
	}
	resp := make([]models.AgentContextResponse, 0, len(list))
	for _, c := range list {
		payloadMap := map[string]any{}
		if c.Payload != "" {
			_ = json.Unmarshal([]byte(c.Payload), &payloadMap)
		}
		tagsList := []string{}
		if c.Tags != "" {
			tagsList = strings.Split(c.Tags, ",")
			for i := range tagsList {
				tagsList[i] = strings.TrimSpace(tagsList[i])
			}
		}
		resp = append(resp, models.AgentContextResponse{
			ID:             c.ID,
			AppID:          c.AppID,
			ExternalUserID: c.ExternalUserID,
			AgentID:        c.AgentID,
			MemoryType:     c.MemoryType,
			Payload:        payloadMap,
			Tags:           tagsList,
			CreatedAt:      c.CreatedAt,
			UpdatedAt:      c.UpdatedAt,
		})
	}
	helpers.WriteJSON(w, http.StatusOK, resp)
}

// HandleGetAgentContext returns one agent context by ID (Neutron-compatible)
func (h *Handlers) HandleGetAgentContext(w http.ResponseWriter, r *http.Request) {
	id, ok := helpers.ExtractAndParseID(w, r.URL.Path, "/agent-contexts/")
	if !ok {
		return
	}
	ctx, err := h.store.GetAgentContextByID(id)
	if err != nil {
		if helpers.HandleNotFoundError(w, err, "Agent context") {
			return
		}
		helpers.HandleInternalErrorSlog(w, "get agent context error", "error", err, "id", id)
		return
	}
	payloadMap := map[string]any{}
	if ctx.Payload != "" {
		_ = json.Unmarshal([]byte(ctx.Payload), &payloadMap)
	}
	tagsList := []string{}
	if ctx.Tags != "" {
		tagsList = strings.Split(ctx.Tags, ",")
		for i := range tagsList {
			tagsList[i] = strings.TrimSpace(tagsList[i])
		}
	}
	helpers.WriteJSON(w, http.StatusOK, models.AgentContextResponse{
		ID:             ctx.ID,
		AppID:          ctx.AppID,
		ExternalUserID: ctx.ExternalUserID,
		AgentID:        ctx.AgentID,
		MemoryType:     ctx.MemoryType,
		Payload:        payloadMap,
		Tags:           tagsList,
		CreatedAt:      ctx.CreatedAt,
		UpdatedAt:      ctx.UpdatedAt,
	})
}
