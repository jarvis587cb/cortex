package store

import (
	"testing"

	"cortex/internal/models"
)

func TestGetAnalytics(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	appID := "testapp"
	userID := "user1"

	// Create test data
	mem1 := &models.Memory{
		Type:           "semantic",
		Content:        "Test memory 1",
		AppID:          appID,
		ExternalUserID: userID,
		Importance:     5,
		Embedding:      "test-embedding",
	}
	mem2 := &models.Memory{
		Type:           "episodic",
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

	// Get analytics
	analytics, err := store.GetAnalytics(appID, userID, 30)
	if err != nil {
		t.Fatalf("GetAnalytics failed: %v", err)
	}

	if analytics.TotalMemories != 2 {
		t.Errorf("expected 2 memories, got %d", analytics.TotalMemories)
	}

	if analytics.TotalBundles != 1 {
		t.Errorf("expected 1 bundle, got %d", analytics.TotalBundles)
	}

	if analytics.MemoriesWithEmbeddings != 1 {
		t.Errorf("expected 1 memory with embedding, got %d", analytics.MemoriesWithEmbeddings)
	}

	if analytics.MemoriesByType["semantic"] != 1 {
		t.Errorf("expected 1 semantic memory, got %d", analytics.MemoriesByType["semantic"])
	}

	if analytics.MemoriesByType["episodic"] != 1 {
		t.Errorf("expected 1 episodic memory, got %d", analytics.MemoriesByType["episodic"])
	}
}

func TestGetGlobalAnalytics(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Create test data
	mem := &models.Memory{
		Type:       "semantic",
		Content:    "Test memory",
		Importance: 5,
	}
	store.CreateMemory(mem)

	bundle := &models.Bundle{
		Name:           "Test Bundle",
		AppID:          "app1",
		ExternalUserID: "user1",
	}
	store.CreateBundle(bundle)

	// Get global analytics
	analytics, err := store.GetGlobalAnalytics(30)
	if err != nil {
		t.Fatalf("GetGlobalAnalytics failed: %v", err)
	}

	if analytics.TotalMemories < 1 {
		t.Errorf("expected at least 1 memory, got %d", analytics.TotalMemories)
	}

	if analytics.TotalBundles < 1 {
		t.Errorf("expected at least 1 bundle, got %d", analytics.TotalBundles)
	}

	if analytics.TenantID != "global" {
		t.Errorf("expected tenant_id 'global', got %s", analytics.TenantID)
	}
}
