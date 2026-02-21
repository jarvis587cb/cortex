package store

import (
	"os"
	"path/filepath"
	"testing"

	"cortex/internal/models"
)

func TestExportImport(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Create test data
	appID := "testapp"
	userID := "user1"

	mem1 := &models.Memory{
		Type:           "semantic",
		Content:        "Test memory 1",
		AppID:          appID,
		ExternalUserID: userID,
		Importance:     5,
	}
	mem2 := &models.Memory{
		Type:           "semantic",
		Content:        "Test memory 2",
		AppID:          appID,
		ExternalUserID: userID,
		Importance:     5,
	}
	store.CreateMemory(mem1)
	store.CreateMemory(mem2)

	bundle := &models.Bundle{
		Name:           "Test Bundle",
		AppID:          appID,
		ExternalUserID: userID,
	}
	store.CreateBundle(bundle)

	// Export data
	exportData, err := store.ExportAll(appID, userID, false)
	if err != nil {
		t.Fatalf("ExportAll failed: %v", err)
	}

	if len(exportData.Memories) != 2 {
		t.Errorf("expected 2 memories, got %d", len(exportData.Memories))
	}

	if len(exportData.Bundles) != 1 {
		t.Errorf("expected 1 bundle, got %d", len(exportData.Bundles))
	}

	// Create new store for import test
	store2 := setupTestDB(t)
	defer store2.Close()

	// Import data
	if err := store2.ImportData(exportData, false); err != nil {
		t.Fatalf("ImportData failed: %v", err)
	}

	// Verify imported data
	importedMemories, _ := store2.ExportMemories(appID, userID, false)
	if len(importedMemories) != 2 {
		t.Errorf("expected 2 imported memories, got %d", len(importedMemories))
	}

	importedBundles, _ := store2.ExportBundles(appID, userID)
	if len(importedBundles) != 1 {
		t.Errorf("expected 1 imported bundle, got %d", len(importedBundles))
	}
}

func TestBackupDatabase(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Create test data
	mem := &models.Memory{
		Type:       "semantic",
		Content:    "Test memory",
		Importance: 5,
	}
	store.CreateMemory(mem)

	// Create backup
	tmpDir := t.TempDir()
	backupPath := filepath.Join(tmpDir, "backup.db")

	if err := store.BackupDatabase(backupPath); err != nil {
		t.Fatalf("BackupDatabase failed: %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("backup file was not created")
	}
}

func TestGetDatabasePath(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	path, err := store.GetDatabasePath()
	if err != nil {
		t.Fatalf("GetDatabasePath failed: %v", err)
	}

	if path == "" {
		t.Error("database path should not be empty")
	}
}
