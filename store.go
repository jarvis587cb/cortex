package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
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
		dbPath = filepath.Join(home, ".openclaw", DefaultDBName)
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
	return s.db.AutoMigrate(&Memory{}, &Entity{}, &Relation{})
}

func (s *CortexStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Memory Operations

func (s *CortexStore) CreateMemory(mem *Memory) error {
	return s.db.Create(mem).Error
}

func (s *CortexStore) SearchMemories(query, memType string, limit int) ([]Memory, error) {
	var memories []Memory
	dbQuery := s.db.Model(&Memory{})

	if query != "" {
		dbQuery = dbQuery.Where("content LIKE ? OR tags LIKE ?", "%"+query+"%", "%"+query+"%")
	}
	if memType != "" {
		dbQuery = dbQuery.Where("type = ?", memType)
	}

	err := dbQuery.Order("created_at DESC").Limit(limit).Find(&memories).Error
	return memories, err
}

func (s *CortexStore) SearchMemoriesByTenant(appID, externalUserID, query string, limit int) ([]Memory, error) {
	var memories []Memory
	dbQuery := s.db.Model(&Memory{}).
		Where("app_id = ? AND external_user_id = ?", appID, externalUserID).
		Where("content LIKE ?", "%"+query+"%")

	err := dbQuery.Order("created_at DESC").Limit(limit).Find(&memories).Error
	return memories, err
}

func (s *CortexStore) GetMemoryByID(id int64) (*Memory, error) {
	var mem Memory
	err := s.db.First(&mem, id).Error
	if err != nil {
		return nil, err
	}
	return &mem, nil
}

func (s *CortexStore) GetMemoryByIDAndTenant(id int64, appID, externalUserID string) (*Memory, error) {
	var mem Memory
	err := s.db.Where("id = ? AND app_id = ? AND external_user_id = ?", id, appID, externalUserID).First(&mem).Error
	if err != nil {
		return nil, err
	}
	return &mem, nil
}

func (s *CortexStore) DeleteMemory(mem *Memory) error {
	return s.db.Delete(mem).Error
}

// Entity Operations

func (s *CortexStore) GetEntity(name string) (*Entity, error) {
	var ent Entity
	err := s.db.Where("name = ?", name).First(&ent).Error
	if err != nil {
		return nil, err
	}
	return &ent, nil
}

func (s *CortexStore) ListEntities() ([]Entity, error) {
	var entities []Entity
	err := s.db.Order("updated_at DESC").Find(&entities).Error
	return entities, err
}

func (s *CortexStore) UpdateEntity(ent *Entity) error {
	ent.UpdatedAt = time.Now()
	return s.db.Save(ent).Error
}

func (s *CortexStore) CreateOrUpdateEntity(ent *Entity) error {
	ent.UpdatedAt = time.Now()
	return s.db.Save(ent).Error
}

// Relation Operations

func (s *CortexStore) GetRelations(entity string) ([]Relation, error) {
	var relations []Relation
	dbQuery := s.db.Model(&Relation{})

	if entity != "" {
		dbQuery = dbQuery.Where("from_entity = ? OR to_entity = ?", entity, entity)
	}

	err := dbQuery.Order("created_at DESC").Find(&relations).Error
	return relations, err
}

func (s *CortexStore) CreateOrUpdateRelation(rel *Relation) error {
	var existing Relation
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

func (s *CortexStore) GetStats() (*Stats, error) {
	var stats Stats
	if err := s.db.Model(&Memory{}).Count(&stats.Memories).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&Entity{}).Count(&stats.Entities).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&Relation{}).Count(&stats.Relations).Error; err != nil {
		return nil, err
	}
	return &stats, nil
}
