package store

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"cortex/internal/embeddings"
	"cortex/internal/helpers"
	"cortex/internal/models"
)

type CortexStore struct {
	db *gorm.DB
}

// GetDB returns the underlying GORM database connection (for transactions)
func (s *CortexStore) GetDB() *gorm.DB {
	return s.db
}

func NewCortexStore(dbPath string) (*CortexStore, error) {
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dbPath = filepath.Join(home, ".openclaw", helpers.DefaultDBName)
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	store := &CortexStore{db: db}
	if err := store.migrate(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *CortexStore) migrate() error {
	if err := s.db.AutoMigrate(&models.Memory{}, &models.MemoryVersion{}, &models.Entity{}, &models.Relation{}, &models.Bundle{}, &models.Webhook{}, &models.AgentContext{}); err != nil {
		return err
	}

	// Backfill status for existing memories (pre-TTL schema)
	s.db.Exec("UPDATE memories SET status = ? WHERE status = '' OR status IS NULL", models.MemoryStatusActive)

	// Composite Indizes für häufigste Queries
	for _, q := range []string{
		"CREATE INDEX IF NOT EXISTS idx_memory_tenant ON memories(app_id, external_user_id)",
		"CREATE INDEX IF NOT EXISTS idx_memory_tenant_bundle ON memories(app_id, external_user_id, bundle_id)",
		"CREATE INDEX IF NOT EXISTS idx_memory_created_at ON memories(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_memory_embedding ON memories(embedding) WHERE embedding != '' AND embedding IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_memory_status ON memories(status)",
		"CREATE INDEX IF NOT EXISTS idx_memory_expires_at ON memories(expires_at) WHERE expires_at IS NOT NULL",
	} {
		if err := s.db.Exec(q).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *CortexStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// applyTenantFilter applies tenant filter (app_id and external_user_id) to a query
func (s *CortexStore) applyTenantFilter(dbQuery *gorm.DB, appID, externalUserID string) *gorm.DB {
	return dbQuery.Where("app_id = ? AND external_user_id = ?", appID, externalUserID)
}

// applyOptionalFilters applies optional filters to a query based on a filter map
func (s *CortexStore) applyOptionalFilters(dbQuery *gorm.DB, filters map[string]interface{}) *gorm.DB {
	if query, ok := filters["query"].(string); ok && query != "" {
		dbQuery = dbQuery.Where("content LIKE ?", "%"+query+"%")
	}
	if memType, ok := filters["memType"].(string); ok && memType != "" {
		dbQuery = dbQuery.Where("type = ?", memType)
	}
	if bundleID, ok := filters["bundleID"].(*int64); ok && bundleID != nil {
		dbQuery = dbQuery.Where("bundle_id = ?", *bundleID)
	}
	if entity, ok := filters["entity"].(string); ok && entity != "" {
		dbQuery = dbQuery.Where("from_entity = ? OR to_entity = ?", entity, entity)
	}
	if appID, ok := filters["appID"].(string); ok && appID != "" {
		dbQuery = dbQuery.Where("app_id = ? OR app_id = ?", appID, "")
	}
	if seedIDs, ok := filters["seedIDs"].([]int64); ok && len(seedIDs) > 0 {
		dbQuery = dbQuery.Where("id IN ?", seedIDs)
	}
	// Metadata filter: filter by JSON fields in metadata column using SQLite JSON1 extension
	if metadataFilter, ok := filters["metadataFilter"].(map[string]any); ok && len(metadataFilter) > 0 {
		for key, value := range metadataFilter {
			if !helpers.SafeJSONPathKey(key) {
				continue
			}
			// Use json_extract to query JSON fields in SQLite
			// Handle both string and other types
			if strValue, isString := value.(string); isString {
				dbQuery = dbQuery.Where("json_extract(metadata, ?) = ?", "$."+key, strValue)
			} else {
				dbQuery = dbQuery.Where("json_extract(metadata, ?) = ?", "$."+key, value)
			}
		}
	}
	return dbQuery
}

// memoryStatusFilter applies status filter unless includeArchived is true.
func (s *CortexStore) memoryStatusFilter(dbQuery *gorm.DB, includeArchived bool) *gorm.DB {
	if !includeArchived {
		return dbQuery.Where("status = ?", models.MemoryStatusActive)
	}
	return dbQuery
}

// Memory Operations

func (s *CortexStore) CreateMemory(mem *models.Memory) error {
	if mem.Status == "" {
		mem.Status = models.MemoryStatusActive
	}
	return s.db.Create(mem).Error
}

func (s *CortexStore) SearchMemories(query, memType string, limit int) ([]models.Memory, error) {
	var memories []models.Memory
	dbQuery := s.db.Model(&models.Memory{})

	filters := map[string]interface{}{
		"query":   query,
		"memType": memType,
	}
	dbQuery = s.applyOptionalFilters(dbQuery, filters)

	// Special handling for tags search
	if query != "" {
		dbQuery = dbQuery.Or("tags LIKE ?", "%"+query+"%")
	}

	err := dbQuery.Order("created_at DESC").Limit(limit).Find(&memories).Error
	return memories, err
}

// ListMemoriesByTenant returns memories for a tenant with pagination (for dashboard/admin).
// Does not populate Embedding in results; use for list views only.
// includeArchived: if false, only active memories are returned.
func (s *CortexStore) ListMemoriesByTenant(appID, externalUserID string, limit, offset int, includeArchived bool) ([]models.Memory, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var memories []models.Memory
	dbQuery := s.applyTenantFilter(s.db.Model(&models.Memory{}), appID, externalUserID)
	dbQuery = s.memoryStatusFilter(dbQuery, includeArchived)
	err := dbQuery.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&memories).Error
	if err != nil {
		return nil, err
	}
	// Clear embedding for list response (not needed, reduces payload)
	for i := range memories {
		memories[i].Embedding = ""
	}
	return memories, nil
}

func (s *CortexStore) SearchMemoriesByTenantAndBundle(appID, externalUserID, query string, bundleID *int64, limit int, seedIDs []int64, metadataFilter map[string]any, includeArchived bool) ([]models.Memory, error) {
	var memories []models.Memory
	dbQuery := s.applyTenantFilter(s.db.Model(&models.Memory{}), appID, externalUserID)
	dbQuery = s.memoryStatusFilter(dbQuery, includeArchived)

	filters := map[string]interface{}{
		"query":          query,
		"bundleID":       bundleID,
		"seedIDs":        seedIDs,
		"metadataFilter": metadataFilter,
	}
	dbQuery = s.applyOptionalFilters(dbQuery, filters)

	err := dbQuery.Order("created_at DESC").Limit(limit).Find(&memories).Error
	return memories, err
}

// SearchMemoriesByTenantSemantic führt semantische Suche mit Embeddings durch
func (s *CortexStore) SearchMemoriesByTenantSemanticAndBundle(appID, externalUserID, query string, bundleID *int64, limit int, seedIDs []int64, metadataFilter map[string]any, includeArchived bool) ([]models.Memory, error) {
	// Generiere Embedding für Query
	embeddingService := embeddings.GetEmbeddingService()
	queryEmbedding, err := embeddingService.GenerateEmbedding(query, "text/plain")
	if err != nil {
		// Fallback zu Textsuche bei Fehler
		return s.SearchMemoriesByTenantAndBundle(appID, externalUserID, query, bundleID, limit, seedIDs, metadataFilter, includeArchived)
	}

	if queryEmbedding == nil {
		// Fallback zu Textsuche wenn kein Embedding generiert werden konnte
		return s.SearchMemoriesByTenantAndBundle(appID, externalUserID, query, bundleID, limit, seedIDs, metadataFilter, includeArchived)
	}

	// Hole alle Memories für diesen Tenant (und optional Bundle, optional seedIDs, optional metadataFilter)
	var allMemories []models.Memory
	dbQuery := s.applyTenantFilter(s.db.Model(&models.Memory{}), appID, externalUserID)
	dbQuery = s.memoryStatusFilter(dbQuery, includeArchived)
	if bundleID != nil {
		dbQuery = dbQuery.Where("bundle_id = ?", *bundleID)
	}
	if len(seedIDs) > 0 {
		dbQuery = dbQuery.Where("id IN ?", seedIDs)
	}
	// Apply metadata filter if provided
	if len(metadataFilter) > 0 {
		filters := map[string]interface{}{
			"metadataFilter": metadataFilter,
		}
		dbQuery = s.applyOptionalFilters(dbQuery, filters)
	}
	err = dbQuery.Find(&allMemories).Error
	if err != nil {
		return nil, err
	}

	// Berechne Similarity für jedes Memory
	type memoryWithSimilarity struct {
		memory     models.Memory
		similarity float64
	}

	results := make([]memoryWithSimilarity, 0, len(allMemories))
	for _, mem := range allMemories {
		if mem.Embedding == "" {
			// Skip Memories ohne Embedding (können später generiert werden)
			continue
		}

		memEmbedding, err := embeddings.DecodeVector(mem.Embedding)
		if err != nil {
			continue
		}

		similarity := embeddings.CosineSimilarity(queryEmbedding, memEmbedding)
		results = append(results, memoryWithSimilarity{
			memory:     mem,
			similarity: similarity,
		})
	}

	// Sortiere nach Similarity (höchste zuerst)
	sort.Slice(results, func(i, j int) bool {
		return results[i].similarity > results[j].similarity
	})

	// Limitiere Ergebnisse
	if limit > len(results) {
		limit = len(results)
	}

	memories := make([]models.Memory, limit)
	for i := 0; i < limit; i++ {
		memories[i] = results[i].memory
	}

	return memories, nil
}

// GenerateEmbeddingForMemory generiert ein Embedding für ein Memory
func (s *CortexStore) GenerateEmbeddingForMemory(mem *models.Memory) error {
	embeddingService := embeddings.GetEmbeddingService()

	// Bestimme Content-Type
	metadata := helpers.UnmarshalMetadata(mem.Metadata)
	contentType := embeddings.DetectContentType(mem.Content, metadata)

	// Generiere Embedding
	embedding, err := embeddingService.GenerateEmbedding(mem.Content, contentType)
	if err != nil {
		return err
	}

	// Speichere Embedding
	embeddingJSON, err := embeddings.EncodeVector(embedding)
	if err != nil {
		return err
	}

	mem.Embedding = embeddingJSON
	mem.ContentType = contentType

	return s.db.Save(mem).Error
}

// BatchGenerateEmbeddings generiert Embeddings für alle Memories ohne Embedding
func (s *CortexStore) BatchGenerateEmbeddings(batchSize int) error {
	if batchSize <= 0 {
		batchSize = 10
	}

	var memories []models.Memory
	err := s.db.Where("embedding = '' OR embedding IS NULL").
		Limit(batchSize).
		Find(&memories).Error
	if err != nil {
		return err
	}

	for i := range memories {
		if err := s.GenerateEmbeddingForMemory(&memories[i]); err != nil {
			// Logge Fehler, aber fahre fort
			continue
		}
	}

	return nil
}

func (s *CortexStore) GetMemoryByIDAndTenant(id int64, appID, externalUserID string, includeArchived bool) (*models.Memory, error) {
	var mem models.Memory
	dbQuery := s.applyTenantFilter(s.db.Model(&models.Memory{}), appID, externalUserID).
		Where("id = ?", id)
	dbQuery = s.memoryStatusFilter(dbQuery, includeArchived)
	err := dbQuery.First(&mem).Error
	if err != nil {
		return nil, err
	}
	return &mem, nil
}

func (s *CortexStore) DeleteMemory(mem *models.Memory) error {
	return s.db.Delete(mem).Error
}

// UpdateMemory updates a memory (tenant must match). Before update, a snapshot is written to memory_versions.
// changedBy can be "api", "merge", "import", etc.
func (s *CortexStore) UpdateMemory(mem *models.Memory, changedBy string) error {
	var existing models.Memory
	err := s.applyTenantFilter(s.db.Model(&models.Memory{}), mem.AppID, mem.ExternalUserID).
		Where("id = ?", mem.ID).First(&existing).Error
	if err != nil {
		return err
	}
	// Next version number
	var maxVersion int
	s.db.Model(&models.MemoryVersion{}).Where("memory_id = ?", mem.ID).Select("COALESCE(MAX(version), 0)").Scan(&maxVersion)
	nextVersion := maxVersion + 1
	// Snapshot current state into memory_versions
	ver := models.MemoryVersion{
		MemoryID:  existing.ID,
		Version:   nextVersion,
		Content:   existing.Content,
		Metadata:  existing.Metadata,
		Importance: existing.Importance,
		Tags:      existing.Tags,
		Entity:    existing.Entity,
		Type:      existing.Type,
		ChangedBy: changedBy,
	}
	if err := s.db.Create(&ver).Error; err != nil {
		return err
	}
	now := time.Now()
	mem.UpdatedAt = &now
	return s.db.Save(mem).Error
}

// ListMemoryVersions returns version history for a memory (tenant-scoped).
func (s *CortexStore) ListMemoryVersions(memoryID int64, appID, externalUserID string) ([]models.MemoryVersion, error) {
	var mem models.Memory
	err := s.applyTenantFilter(s.db.Model(&models.Memory{}), appID, externalUserID).
		Where("id = ?", memoryID).First(&mem).Error
	if err != nil {
		return nil, err
	}
	var versions []models.MemoryVersion
	err = s.db.Where("memory_id = ?", memoryID).Order("version DESC").Find(&versions).Error
	return versions, err
}

// ArchiveMemoriesByExpiry sets status to archived for all active memories where expires_at <= until.
// Returns the number of rows updated.
func (s *CortexStore) ArchiveMemoriesByExpiry(until time.Time) (int64, error) {
	res := s.db.Model(&models.Memory{}).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at <= ?", models.MemoryStatusActive, until).
		Update("status", models.MemoryStatusArchived)
	return res.RowsAffected, res.Error
}

// DeleteArchivedOlderThan permanently deletes archived memories whose updated_at (or created_at) is before cutoff.
// Uses created_at when updated_at is NULL. Returns the number of rows deleted.
func (s *CortexStore) DeleteArchivedOlderThan(cutoff time.Time) (int64, error) {
	// SQLite: delete where status=archived and (updated_at < cutoff or (updated_at is null and created_at < cutoff))
	res := s.db.Where("status = ?", models.MemoryStatusArchived).
		Where("(updated_at IS NOT NULL AND updated_at < ?) OR (updated_at IS NULL AND created_at < ?)", cutoff, cutoff).
		Delete(&models.Memory{})
	return res.RowsAffected, res.Error
}

// FindSimilarMemoryPairs returns pairs of memory IDs (keepID, mergeID) that have similarity >= minSimilarity.
// Only active memories with embeddings are considered. Optionally scoped by bundleID.
func (s *CortexStore) FindSimilarMemoryPairs(appID, externalUserID string, bundleID *int64, minSimilarity float64, limit int) ([][2]int64, error) {
	var memories []models.Memory
	dbQuery := s.applyTenantFilter(s.db.Model(&models.Memory{}), appID, externalUserID).
		Where("status = ? AND embedding != '' AND embedding IS NOT NULL", models.MemoryStatusActive)
	if bundleID != nil {
		dbQuery = dbQuery.Where("bundle_id = ?", *bundleID)
	}
	if err := dbQuery.Find(&memories).Error; err != nil {
		return nil, err
	}
	type pair struct{ a, b int64 }
	seen := make(map[pair]struct{})
	var result [][2]int64
	for i := range memories {
		for j := i + 1; j < len(memories); j++ {
			if limit > 0 && len(result) >= limit {
				return result, nil
			}
			embA, _ := embeddings.DecodeVector(memories[i].Embedding)
			embB, _ := embeddings.DecodeVector(memories[j].Embedding)
			if embA == nil || embB == nil {
				continue
			}
			sim := embeddings.CosineSimilarity(embA, embB)
			if sim < minSimilarity {
				continue
			}
			// Keep lower ID first (deterministic: we merge into the older one)
			idA, idB := memories[i].ID, memories[j].ID
			if idA > idB {
				idA, idB = idB, idA
			}
			p := pair{idA, idB}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			result = append(result, [2]int64{idA, idB})
		}
	}
	return result, nil
}

// MergeMemories merges the "merge" memory into the "keep" memory (both must exist and belong to tenant).
// Content is concatenated with " | "; metadata/tags are merged; importance = max. Embedding is regenerated for keep.
// The merged memory is archived and its metadata gets merged_into = keep.ID.
func (s *CortexStore) MergeMemories(keepID, mergeID int64, appID, externalUserID string) error {
	keep, err := s.GetMemoryByIDAndTenant(keepID, appID, externalUserID, false)
	if err != nil {
		return err
	}
	merge, err := s.GetMemoryByIDAndTenant(mergeID, appID, externalUserID, false)
	if err != nil {
		return err
	}
	// Merge content (concatenate with separator)
	mergedContent := keep.Content
	if merge.Content != "" {
		if mergedContent != "" {
			mergedContent += " | " + merge.Content
		} else {
			mergedContent = merge.Content
		}
	}
	keep.Content = mergedContent
	// Merge metadata: merge keys into keep's metadata
	metaKeep := helpers.UnmarshalMetadata(keep.Metadata)
	metaMerge := helpers.UnmarshalMetadata(merge.Metadata)
	for k, v := range metaMerge {
		if _, exists := metaKeep[k]; !exists {
			metaKeep[k] = v
		}
	}
	keep.Metadata = helpers.MarshalMetadata(metaKeep)
	// Tags: combine
	tagsKeep := strings.Split(keep.Tags, ",")
	tagSet := make(map[string]struct{})
	for _, t := range tagsKeep {
		tagSet[strings.TrimSpace(t)] = struct{}{}
	}
	for _, t := range strings.Split(merge.Tags, ",") {
		tagSet[strings.TrimSpace(t)] = struct{}{}
	}
	var tagList []string
	for t := range tagSet {
		if t != "" {
			tagList = append(tagList, t)
		}
	}
	keep.Tags = strings.Join(tagList, ",")
	// Importance: max
	if merge.Importance > keep.Importance {
		keep.Importance = merge.Importance
	}
	if err := s.UpdateMemory(keep, "merge"); err != nil {
		return err
	}
	if err := s.GenerateEmbeddingForMemory(keep); err != nil {
		// non-fatal
		_ = err
	}
	// Archive merged memory and set merged_into in metadata
	mergeMeta := helpers.UnmarshalMetadata(merge.Metadata)
	mergeMeta["merged_into"] = float64(keepID) // JSON numbers are float64
	merge.Metadata = helpers.MarshalMetadata(mergeMeta)
	merge.Status = models.MemoryStatusArchived
	now := time.Now()
	merge.UpdatedAt = &now
	return s.db.Save(merge).Error
}

// Entity Operations

func (s *CortexStore) GetEntity(name string) (*models.Entity, error) {
	var ent models.Entity
	err := s.db.Where("name = ?", name).First(&ent).Error
	if err != nil {
		return nil, err
	}
	return &ent, nil
}

func (s *CortexStore) ListEntities() ([]models.Entity, error) {
	var entities []models.Entity
	err := s.db.Order("updated_at DESC").Find(&entities).Error
	return entities, err
}

func (s *CortexStore) CreateOrUpdateEntity(ent *models.Entity) error {
	ent.UpdatedAt = time.Now()
	
	// Use ON CONFLICT to handle race conditions atomically
	// This ensures that concurrent requests don't cause UNIQUE constraint errors
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"data", "updated_at"}),
	}).Create(ent).Error
}

// Relation Operations

func (s *CortexStore) GetRelations(entity string) ([]models.Relation, error) {
	var relations []models.Relation
	dbQuery := s.db.Model(&models.Relation{})

	filters := map[string]interface{}{
		"entity": entity,
	}
	dbQuery = s.applyOptionalFilters(dbQuery, filters)

	err := dbQuery.Order("created_at DESC").Find(&relations).Error
	return relations, err
}

func (s *CortexStore) CreateOrUpdateRelation(rel *models.Relation) error {
	var existing models.Relation
	result := s.db.Where("from_entity = ? AND to_entity = ? AND type = ?", rel.From, rel.To, rel.Type).First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		return s.db.Create(rel).Error
	} else if result.Error != nil {
		return result.Error
	}

	existing.From = rel.From
	existing.To = rel.To
	existing.Type = rel.Type
	return s.db.Save(&existing).Error
}

// Stats

func (s *CortexStore) GetStats() (*models.Stats, error) {
	var stats models.Stats
	var results struct {
		Memories  int64 `gorm:"column:memories"`
		Entities  int64 `gorm:"column:entities"`
		Relations int64 `gorm:"column:relations"`
	}

	err := s.db.Raw(`
		SELECT 
			(SELECT COUNT(*) FROM memories) as memories,
			(SELECT COUNT(*) FROM entities) as entities,
			(SELECT COUNT(*) FROM relations) as relations
	`).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	stats.Memories = results.Memories
	stats.Entities = results.Entities
	stats.Relations = results.Relations

	return &stats, nil
}

// Bundle Operations

func (s *CortexStore) CreateBundle(bundle *models.Bundle) error {
	return s.db.Create(bundle).Error
}

func (s *CortexStore) GetBundle(id int64, appID, externalUserID string) (*models.Bundle, error) {
	var bundle models.Bundle
	err := s.applyTenantFilter(s.db.Model(&models.Bundle{}), appID, externalUserID).
		Where("id = ?", id).
		First(&bundle).Error
	if err != nil {
		return nil, err
	}
	return &bundle, nil
}

func (s *CortexStore) ListBundles(appID, externalUserID string) ([]models.Bundle, error) {
	var bundles []models.Bundle
	err := s.applyTenantFilter(s.db.Model(&models.Bundle{}), appID, externalUserID).
		Order("created_at DESC").
		Find(&bundles).Error
	return bundles, err
}

func (s *CortexStore) DeleteBundle(id int64, appID, externalUserID string) error {
	// Setze bundle_id auf NULL für alle Memories in diesem Bundle
	if err := s.applyTenantFilter(s.db.Model(&models.Memory{}), appID, externalUserID).
		Where("bundle_id = ?", id).
		Update("bundle_id", nil).Error; err != nil {
		return err
	}

	// Lösche das Bundle
	return s.applyTenantFilter(s.db.Model(&models.Bundle{}), appID, externalUserID).
		Where("id = ?", id).
		Delete(&models.Bundle{}).Error
}

// Webhook Operations

func (s *CortexStore) CreateWebhook(webhook *models.Webhook) error {
	return s.db.Create(webhook).Error
}

func (s *CortexStore) GetWebhook(id int64) (*models.Webhook, error) {
	var webhook models.Webhook
	err := s.db.First(&webhook, id).Error
	if err != nil {
		return nil, err
	}
	return &webhook, nil
}

func (s *CortexStore) ListWebhooks(appID string) ([]models.Webhook, error) {
	var webhooks []models.Webhook
	dbQuery := s.db.Model(&models.Webhook{}).Where("active = ?", true)

	filters := map[string]interface{}{
		"appID": appID,
	}
	dbQuery = s.applyOptionalFilters(dbQuery, filters)

	err := dbQuery.Order("created_at DESC").Find(&webhooks).Error
	return webhooks, err
}

func (s *CortexStore) UpdateWebhook(webhook *models.Webhook) error {
	webhook.UpdatedAt = time.Now()
	return s.db.Save(webhook).Error
}

func (s *CortexStore) DeleteWebhook(id int64) error {
	return s.db.Delete(&models.Webhook{}, id).Error
}

// GetWebhookByIDAndApp returns a webhook only if it belongs to the given app (tenant isolation).
func (s *CortexStore) GetWebhookByIDAndApp(id int64, appID string) (*models.Webhook, error) {
	var wh models.Webhook
	err := s.db.Where("id = ? AND app_id = ?", id, appID).First(&wh).Error
	if err != nil {
		return nil, err
	}
	return &wh, nil
}

// Agent Contexts (Neutron-compatible)

func (s *CortexStore) CreateAgentContext(ctx *models.AgentContext) error {
	return s.db.Create(ctx).Error
}

func (s *CortexStore) ListAgentContexts(appID, externalUserID, agentID, memoryType, tagsFilter string) ([]models.AgentContext, error) {
	var list []models.AgentContext
	dbQuery := s.db.Model(&models.AgentContext{}).Where("app_id = ? AND external_user_id = ?", appID, externalUserID)
	if agentID != "" {
		dbQuery = dbQuery.Where("agent_id = ?", agentID)
	}
	if memoryType != "" {
		dbQuery = dbQuery.Where("memory_type = ?", memoryType)
	}
	if tagsFilter != "" {
		dbQuery = dbQuery.Where("tags LIKE ?", "%"+tagsFilter+"%")
	}
	err := dbQuery.Order("updated_at DESC").Find(&list).Error
	return list, err
}

func (s *CortexStore) GetAgentContextByID(id int64) (*models.AgentContext, error) {
	var ctx models.AgentContext
	err := s.db.First(&ctx, id).Error
	if err != nil {
		return nil, err
	}
	return &ctx, nil
}

// GetAgentContextByIDAndTenant returns an agent context only if it belongs to the given tenant.
func (s *CortexStore) GetAgentContextByIDAndTenant(id int64, appID, externalUserID string) (*models.AgentContext, error) {
	var ctx models.AgentContext
	err := s.db.Where("id = ? AND app_id = ? AND external_user_id = ?", id, appID, externalUserID).First(&ctx).Error
	if err != nil {
		return nil, err
	}
	return &ctx, nil
}
