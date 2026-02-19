package store

import (
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"cortex/internal/embeddings"
	"cortex/internal/helpers"
	"cortex/internal/models"
)

type CortexStore struct {
	db *gorm.DB
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
	return s.db.AutoMigrate(&models.Memory{}, &models.Entity{}, &models.Relation{}, &models.Bundle{})
}

func (s *CortexStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Memory Operations

func (s *CortexStore) CreateMemory(mem *models.Memory) error {
	return s.db.Create(mem).Error
}

func (s *CortexStore) SearchMemories(query, memType string, limit int) ([]models.Memory, error) {
	var memories []models.Memory
	dbQuery := s.db.Model(&models.Memory{})

	if query != "" {
		dbQuery = dbQuery.Where("content LIKE ? OR tags LIKE ?", "%"+query+"%", "%"+query+"%")
	}
	if memType != "" {
		dbQuery = dbQuery.Where("type = ?", memType)
	}

	err := dbQuery.Order("created_at DESC").Limit(limit).Find(&memories).Error
	return memories, err
}

func (s *CortexStore) SearchMemoriesByTenant(appID, externalUserID, query string, limit int) ([]models.Memory, error) {
	return s.SearchMemoriesByTenantAndBundle(appID, externalUserID, query, nil, limit)
}

func (s *CortexStore) SearchMemoriesByTenantAndBundle(appID, externalUserID, query string, bundleID *int64, limit int) ([]models.Memory, error) {
	var memories []models.Memory
	dbQuery := s.db.Model(&models.Memory{}).
		Where("app_id = ? AND external_user_id = ?", appID, externalUserID)

	if query != "" {
		dbQuery = dbQuery.Where("content LIKE ?", "%"+query+"%")
	}

	if bundleID != nil {
		dbQuery = dbQuery.Where("bundle_id = ?", *bundleID)
	}

	err := dbQuery.Order("created_at DESC").Limit(limit).Find(&memories).Error
	return memories, err
}

// SearchMemoriesByTenantSemantic führt semantische Suche mit Embeddings durch
func (s *CortexStore) SearchMemoriesByTenantSemantic(appID, externalUserID, query string, limit int) ([]models.Memory, error) {
	return s.SearchMemoriesByTenantSemanticAndBundle(appID, externalUserID, query, nil, limit)
}

func (s *CortexStore) SearchMemoriesByTenantSemanticAndBundle(appID, externalUserID, query string, bundleID *int64, limit int) ([]models.Memory, error) {
	// Generiere Embedding für Query
	embeddingService := embeddings.GetEmbeddingService()
	queryEmbedding, err := embeddingService.GenerateEmbedding(query, "text/plain")
	if err != nil {
		// Fallback zu Textsuche bei Fehler
		return s.SearchMemoriesByTenantAndBundle(appID, externalUserID, query, bundleID, limit)
	}

	if queryEmbedding == nil {
		// Fallback zu Textsuche wenn kein Embedding generiert werden konnte
		return s.SearchMemoriesByTenantAndBundle(appID, externalUserID, query, bundleID, limit)
	}

	// Hole alle Memories für diesen Tenant (und optional Bundle)
	var allMemories []models.Memory
	dbQuery := s.db.Model(&models.Memory{}).
		Where("app_id = ? AND external_user_id = ?", appID, externalUserID)
	if bundleID != nil {
		dbQuery = dbQuery.Where("bundle_id = ?", *bundleID)
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
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].similarity < results[j].similarity {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

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

func (s *CortexStore) GetMemoryByID(id int64) (*models.Memory, error) {
	var mem models.Memory
	err := s.db.First(&mem, id).Error
	if err != nil {
		return nil, err
	}
	return &mem, nil
}

func (s *CortexStore) GetMemoryByIDAndTenant(id int64, appID, externalUserID string) (*models.Memory, error) {
	var mem models.Memory
	err := s.db.Where("id = ? AND app_id = ? AND external_user_id = ?", id, appID, externalUserID).First(&mem).Error
	if err != nil {
		return nil, err
	}
	return &mem, nil
}

func (s *CortexStore) DeleteMemory(mem *models.Memory) error {
	return s.db.Delete(mem).Error
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

func (s *CortexStore) UpdateEntity(ent *models.Entity) error {
	ent.UpdatedAt = time.Now()
	return s.db.Save(ent).Error
}

func (s *CortexStore) CreateOrUpdateEntity(ent *models.Entity) error {
	ent.UpdatedAt = time.Now()
	return s.db.Save(ent).Error
}

// Relation Operations

func (s *CortexStore) GetRelations(entity string) ([]models.Relation, error) {
	var relations []models.Relation
	dbQuery := s.db.Model(&models.Relation{})

	if entity != "" {
		dbQuery = dbQuery.Where("from_entity = ? OR to_entity = ?", entity, entity)
	}

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
	if err := s.db.Model(&models.Memory{}).Count(&stats.Memories).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&models.Entity{}).Count(&stats.Entities).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&models.Relation{}).Count(&stats.Relations).Error; err != nil {
		return nil, err
	}
	return &stats, nil
}

// Bundle Operations

func (s *CortexStore) CreateBundle(bundle *models.Bundle) error {
	return s.db.Create(bundle).Error
}

func (s *CortexStore) GetBundle(id int64, appID, externalUserID string) (*models.Bundle, error) {
	var bundle models.Bundle
	err := s.db.Where("id = ? AND app_id = ? AND external_user_id = ?", id, appID, externalUserID).First(&bundle).Error
	if err != nil {
		return nil, err
	}
	return &bundle, nil
}

func (s *CortexStore) ListBundles(appID, externalUserID string) ([]models.Bundle, error) {
	var bundles []models.Bundle
	err := s.db.Where("app_id = ? AND external_user_id = ?", appID, externalUserID).
		Order("created_at DESC").
		Find(&bundles).Error
	return bundles, err
}

func (s *CortexStore) DeleteBundle(id int64, appID, externalUserID string) error {
	// Setze bundle_id auf NULL für alle Memories in diesem Bundle
	if err := s.db.Model(&models.Memory{}).
		Where("bundle_id = ? AND app_id = ? AND external_user_id = ?", id, appID, externalUserID).
		Update("bundle_id", nil).Error; err != nil {
		return err
	}

	// Lösche das Bundle
	return s.db.Where("id = ? AND app_id = ? AND external_user_id = ?", id, appID, externalUserID).
		Delete(&models.Bundle{}).Error
}
