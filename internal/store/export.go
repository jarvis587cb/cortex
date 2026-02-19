package store

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"cortex/internal/models"
)

// ExportData represents exported data structure
type ExportData struct {
	Version    string           `json:"version"`
	ExportDate string           `json:"export_date"`
	Memories   []models.Memory  `json:"memories"`
	Bundles    []models.Bundle  `json:"bundles"`
	Webhooks   []models.Webhook `json:"webhooks"`
}

// ExportMemories exports all memories for a tenant
func (s *CortexStore) ExportMemories(appID, externalUserID string) ([]models.Memory, error) {
	var memories []models.Memory
	err := s.db.Where("app_id = ? AND external_user_id = ?", appID, externalUserID).
		Order("created_at ASC").
		Find(&memories).Error
	return memories, err
}

// ExportBundles exports all bundles for a tenant
func (s *CortexStore) ExportBundles(appID, externalUserID string) ([]models.Bundle, error) {
	var bundles []models.Bundle
	err := s.db.Where("app_id = ? AND external_user_id = ?", appID, externalUserID).
		Order("created_at ASC").
		Find(&bundles).Error
	return bundles, err
}

// ExportWebhooks exports webhooks (optionally filtered by appID)
func (s *CortexStore) ExportWebhooks(appID string) ([]models.Webhook, error) {
	var webhooks []models.Webhook
	query := s.db.Model(&models.Webhook{})
	if appID != "" {
		query = query.Where("app_id = ? OR app_id = ?", appID, "")
	}
	err := query.Order("created_at ASC").Find(&webhooks).Error
	return webhooks, err
}

// ExportAll exports all data for a tenant
func (s *CortexStore) ExportAll(appID, externalUserID string) (*ExportData, error) {
	memories, err := s.ExportMemories(appID, externalUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to export memories: %w", err)
	}

	bundles, err := s.ExportBundles(appID, externalUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to export bundles: %w", err)
	}

	webhooks, err := s.ExportWebhooks(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to export webhooks: %w", err)
	}

	return &ExportData{
		Version:    "1.0",
		ExportDate: time.Now().UTC().Format(time.RFC3339),
		Memories:   memories,
		Bundles:    bundles,
		Webhooks:   webhooks,
	}, nil
}

// ImportMemories imports memories from a slice
func (s *CortexStore) ImportMemories(memories []models.Memory, overwrite bool) error {
	for _, mem := range memories {
		if overwrite && mem.ID > 0 {
			// Update existing memory
			if err := s.db.Save(&mem).Error; err != nil {
				return fmt.Errorf("failed to import memory %d: %w", mem.ID, err)
			}
		} else {
			// Create new memory (ignore ID)
			mem.ID = 0
			if err := s.db.Create(&mem).Error; err != nil {
				return fmt.Errorf("failed to import memory: %w", err)
			}
		}
	}
	return nil
}

// ImportBundles imports bundles from a slice
func (s *CortexStore) ImportBundles(bundles []models.Bundle, overwrite bool) error {
	for _, bundle := range bundles {
		if overwrite && bundle.ID > 0 {
			// Update existing bundle
			if err := s.db.Save(&bundle).Error; err != nil {
				return fmt.Errorf("failed to import bundle %d: %w", bundle.ID, err)
			}
		} else {
			// Create new bundle (ignore ID)
			bundle.ID = 0
			if err := s.db.Create(&bundle).Error; err != nil {
				return fmt.Errorf("failed to import bundle: %w", err)
			}
		}
	}
	return nil
}

// ImportWebhooks imports webhooks from a slice
func (s *CortexStore) ImportWebhooks(webhooks []models.Webhook, overwrite bool) error {
	for _, webhook := range webhooks {
		if overwrite && webhook.ID > 0 {
			// Update existing webhook
			if err := s.db.Save(&webhook).Error; err != nil {
				return fmt.Errorf("failed to import webhook %d: %w", webhook.ID, err)
			}
		} else {
			// Create new webhook (ignore ID)
			webhook.ID = 0
			if err := s.db.Create(&webhook).Error; err != nil {
				return fmt.Errorf("failed to import webhook: %w", err)
			}
		}
	}
	return nil
}

// ImportData imports data from ExportData
func (s *CortexStore) ImportData(data *ExportData, overwrite bool) error {
	if err := s.ImportMemories(data.Memories, overwrite); err != nil {
		return fmt.Errorf("failed to import memories: %w", err)
	}

	if err := s.ImportBundles(data.Bundles, overwrite); err != nil {
		return fmt.Errorf("failed to import bundles: %w", err)
	}

	// Webhooks are imported separately (usually app-level)
	if err := s.ImportWebhooks(data.Webhooks, overwrite); err != nil {
		return fmt.Errorf("failed to import webhooks: %w", err)
	}

	return nil
}

// BackupDatabase creates a backup of the database file
func (s *CortexStore) BackupDatabase(backupPath string) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	// SQLite backup using VACUUM INTO (SQLite 3.27+)
	// Escape single quotes in path
	escapedPath := strings.ReplaceAll(backupPath, "'", "''")
	backupSQL := fmt.Sprintf("VACUUM INTO '%s'", escapedPath)

	// Use raw SQL execution
	_, err = sqlDB.Exec(backupSQL)
	if err != nil {
		return fmt.Errorf("failed to backup database: %w", err)
	}

	return nil
}

// CopyFile copies a file from src to dst
func (s *CortexStore) CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// FileExists checks if a file exists
func (s *CortexStore) FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// GetDatabasePath returns the path to the database file
func (s *CortexStore) GetDatabasePath() (string, error) {
	sqlDB, err := s.db.DB()
	if err != nil {
		return "", fmt.Errorf("failed to get database connection: %w", err)
	}

	var seq int
	var name, path string
	err = sqlDB.QueryRow("PRAGMA database_list").Scan(&seq, &name, &path)
	if err != nil {
		return "", fmt.Errorf("failed to get database path: %w", err)
	}

	return path, nil
}
