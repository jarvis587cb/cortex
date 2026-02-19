package store

import (
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

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
	return s.db.AutoMigrate(&models.Memory{}, &models.Entity{}, &models.Relation{})
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
	var memories []models.Memory
	dbQuery := s.db.Model(&models.Memory{}).
		Where("app_id = ? AND external_user_id = ?", appID, externalUserID).
		Where("content LIKE ?", "%"+query+"%")

	err := dbQuery.Order("created_at DESC").Limit(limit).Find(&memories).Error
	return memories, err
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
