package cleanup

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"cortex/internal/helpers"
	"cortex/internal/models"
	"cortex/internal/store"
)

func setupTestStore(t *testing.T) *store.CortexStore {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	s, err := store.NewCortexStore(dbPath)
	if err != nil {
		t.Fatalf("NewCortexStore: %v", err)
	}
	return s
}

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	if !c.ArchiveByExpiry {
		t.Error("expected ArchiveByExpiry true")
	}
	if c.MergeSimilar {
		t.Error("expected MergeSimilar false")
	}
	if c.MergeMinSimilarity != 0.95 {
		t.Errorf("expected MergeMinSimilarity 0.95, got %v", c.MergeMinSimilarity)
	}
	if c.LowImportanceThreshold != 2 {
		t.Errorf("expected LowImportanceThreshold 2, got %d", c.LowImportanceThreshold)
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("CORTEX_CLEANUP_DRY_RUN", "true")
	t.Setenv("CORTEX_CLEANUP_MERGE_SIMILAR", "1")
	t.Setenv("CORTEX_CLEANUP_MERGE_SIMILARITY", "0.9")
	t.Setenv("CORTEX_CLEANUP_LOW_IMPORTANCE_THRESHOLD", "3")

	c := ConfigFromEnv()
	if !c.DryRun {
		t.Error("expected DryRun true from env")
	}
	if !c.MergeSimilar {
		t.Error("expected MergeSimilar true from env")
	}
	if c.MergeMinSimilarity != 0.9 {
		t.Errorf("expected MergeMinSimilarity 0.9, got %v", c.MergeMinSimilarity)
	}
	if c.LowImportanceThreshold != 3 {
		t.Errorf("expected LowImportanceThreshold 3, got %d", c.LowImportanceThreshold)
	}
}

func TestRunCleanup_ArchiveByExpiry(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()

	expired := time.Now().Add(-time.Minute)
	mem := &models.Memory{
		Type:           "semantic",
		Content:        "Expired",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
		Status:         models.MemoryStatusActive,
		ExpiresAt:      &expired,
	}
	if err := s.CreateMemory(mem); err != nil {
		t.Fatalf("CreateMemory: %v", err)
	}

	cfg := DefaultConfig()
	cfg.DryRun = false
	cfg.ArchiveByExpiry = true

	stats, err := RunCleanup(context.Background(), s, cfg)
	if err != nil {
		t.Fatalf("RunCleanup: %v", err)
	}
	if stats.ArchivedByExpiry != 1 {
		t.Errorf("expected ArchivedByExpiry 1, got %d", stats.ArchivedByExpiry)
	}

	got, _ := s.GetMemoryByIDAndTenant(mem.ID, "app1", "user1", true)
	if got == nil {
		t.Fatal("memory not found")
	}
	if got.Status != models.MemoryStatusArchived {
		t.Errorf("expected status archived, got %s", got.Status)
	}
}

func TestRunCleanup_DryRun(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()

	expired := time.Now().Add(-time.Minute)
	mem := &models.Memory{
		Type:           "semantic",
		Content:        "Expired",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
		Status:         models.MemoryStatusActive,
		ExpiresAt:      &expired,
	}
	if err := s.CreateMemory(mem); err != nil {
		t.Fatalf("CreateMemory: %v", err)
	}

	cfg := DefaultConfig()
	cfg.DryRun = true
	cfg.ArchiveByExpiry = true

	_, err := RunCleanup(context.Background(), s, cfg)
	if err != nil {
		t.Fatalf("RunCleanup: %v", err)
	}

	got, _ := s.GetMemoryByIDAndTenant(mem.ID, "app1", "user1", false)
	if got == nil {
		t.Fatal("memory should still exist and be active (dry run)")
	}
	if got.Status != models.MemoryStatusActive {
		t.Errorf("dry run should not change status, got %s", got.Status)
	}
}

func TestRunCleanup_ArchiveLowImportance(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()

	mem := &models.Memory{
		Type:           "semantic",
		Content:        "Low importance",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     1,
		Status:         models.MemoryStatusActive,
	}
	if err := s.CreateMemory(mem); err != nil {
		t.Fatalf("CreateMemory: %v", err)
	}

	cfg := DefaultConfig()
	cfg.ArchiveLowImportance = true
	cfg.LowImportanceThreshold = 2

	stats, err := RunCleanup(context.Background(), s, cfg)
	if err != nil {
		t.Fatalf("RunCleanup: %v", err)
	}
	if stats.ArchivedLowImport != 1 {
		t.Errorf("expected ArchivedLowImport 1, got %d", stats.ArchivedLowImport)
	}

	got, _ := s.GetMemoryByIDAndTenant(mem.ID, "app1", "user1", true)
	if got == nil {
		t.Fatal("memory not found")
	}
	if got.Status != models.MemoryStatusArchived {
		t.Errorf("expected status archived, got %s", got.Status)
	}
}

func TestRunCleanup_DeleteArchivedOlderThan(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()

	mem := &models.Memory{
		Type:           "semantic",
		Content:        "Old archived",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
		Status:         models.MemoryStatusArchived,
	}
	if err := s.CreateMemory(mem); err != nil {
		t.Fatalf("CreateMemory: %v", err)
	}
	oldTime := time.Now().Add(-48 * time.Hour)
	s.GetDB().Exec("UPDATE memories SET updated_at = ? WHERE id = ?", oldTime, mem.ID)

	cfg := DefaultConfig()
	cfg.DeleteArchivedOlderThan = 24 * time.Hour

	stats, err := RunCleanup(context.Background(), s, cfg)
	if err != nil {
		t.Fatalf("RunCleanup: %v", err)
	}
	if stats.DeletedArchived != 1 {
		t.Errorf("expected DeletedArchived 1, got %d", stats.DeletedArchived)
	}

	_, err = s.GetMemoryByIDAndTenant(mem.ID, "app1", "user1", true)
	if err == nil {
		t.Error("memory should be deleted")
	}
}

// seedCleanupRealData fills the store with realistic data for a full cleanup run:
// expired memories, low-importance memories, and an old archived one (for delete).
func seedCleanupRealData(t *testing.T, s *store.CortexStore) (expiredIDs []int64, lowImportIDs []int64, archivedOldID int64) {
	t.Helper()
	appID, userID := "app1", "user1"
	now := time.Now()
	expired := now.Add(-1 * time.Hour)

	for _, c := range []string{"Abgelaufener Eintrag 1.", "Abgelaufener Eintrag 2.", "Abgelaufener Eintrag 3."} {
		m := &models.Memory{
			Type: "semantic", Content: c, AppID: appID, ExternalUserID: userID,
			Importance: 5, Status: models.MemoryStatusActive, ExpiresAt: &expired,
		}
		if err := s.CreateMemory(m); err != nil {
			t.Fatalf("seed: %v", err)
		}
		expiredIDs = append(expiredIDs, m.ID)
	}
	for _, c := range []string{"Niedrige Priorität 1.", "Niedrige Priorität 2."} {
		m := &models.Memory{
			Type: "semantic", Content: c, AppID: appID, ExternalUserID: userID,
			Importance: 1, Status: models.MemoryStatusActive,
			Metadata: helpers.MarshalMetadata(map[string]any{"source": "test"}),
		}
		if err := s.CreateMemory(m); err != nil {
			t.Fatalf("seed: %v", err)
		}
		lowImportIDs = append(lowImportIDs, m.ID)
	}
	mArch := &models.Memory{
		Type: "semantic", Content: "Bereits archiviert, zum Löschen.", AppID: appID, ExternalUserID: userID,
		Importance: 3, Status: models.MemoryStatusArchived,
	}
	if err := s.CreateMemory(mArch); err != nil {
		t.Fatalf("seed: %v", err)
	}
	oldTime := now.Add(-100 * time.Hour)
	s.GetDB().Exec("UPDATE memories SET updated_at = ? WHERE id = ?", oldTime, mArch.ID)
	archivedOldID = mArch.ID
	return expiredIDs, lowImportIDs, archivedOldID
}

// TestRunCleanup_IntegrationWithRealData runs a full cleanup against a test DB filled with
// realistic data (expired, low-importance, old archived) and asserts on stats and final state.
func TestRunCleanup_IntegrationWithRealData(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()

	expiredIDs, lowImportIDs, archivedOldID := seedCleanupRealData(t, s)

	cfg := Config{
		DryRun:                  false,
		ArchiveByExpiry:         true,
		DeleteArchivedOlderThan: 50 * time.Hour, // our archived is 100h old
		MergeSimilar:            false,
		ArchiveLowImportance:    true,
		LowImportanceThreshold:  2,
	}

	stats, err := RunCleanup(context.Background(), s, cfg)
	if err != nil {
		t.Fatalf("RunCleanup: %v", err)
	}

	if stats.ArchivedByExpiry != 3 {
		t.Errorf("expected ArchivedByExpiry 3, got %d", stats.ArchivedByExpiry)
	}
	if stats.ArchivedLowImport != 2 {
		t.Errorf("expected ArchivedLowImport 2, got %d", stats.ArchivedLowImport)
	}
	if stats.DeletedArchived != 1 {
		t.Errorf("expected DeletedArchived 1, got %d", stats.DeletedArchived)
	}

	for _, id := range expiredIDs {
		got, _ := s.GetMemoryByIDAndTenant(id, "app1", "user1", true)
		if got == nil {
			t.Errorf("memory %d not found", id)
			continue
		}
		if got.Status != models.MemoryStatusArchived {
			t.Errorf("memory %d: expected archived, got %s", id, got.Status)
		}
	}
	for _, id := range lowImportIDs {
		got, _ := s.GetMemoryByIDAndTenant(id, "app1", "user1", true)
		if got == nil {
			t.Errorf("memory %d not found", id)
			continue
		}
		if got.Status != models.MemoryStatusArchived {
			t.Errorf("memory %d: expected archived, got %s", id, got.Status)
		}
	}
	_, err = s.GetMemoryByIDAndTenant(archivedOldID, "app1", "user1", true)
	if err == nil {
		t.Error("archived old memory should be deleted")
	}
}
