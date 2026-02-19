package store

import (
	"os"
	"path/filepath"
	"testing"

	"cortex/internal/helpers"
	"cortex/internal/models"
)

func setupTestDB(t *testing.T) *CortexStore {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCortexStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	return store
}

func TestNewCortexStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewCortexStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	if store.db == nil {
		t.Error("database connection is nil")
	}
}

func TestNewCortexStore_DefaultPath(t *testing.T) {
	// Temporarily set home dir
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)

	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)

	store, err := NewCortexStore("")
	if err != nil {
		t.Fatalf("failed to create store with default path: %v", err)
	}
	defer store.Close()

	expectedPath := filepath.Join(tmpDir, ".openclaw", helpers.DefaultDBName)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("database file not created at expected path: %s", expectedPath)
	}
}

func TestCreateMemory(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	mem := &models.Memory{
		Type:       "semantic",
		Content:    "Test memory",
		Importance: 5,
	}

	err := store.CreateMemory(mem)
	if err != nil {
		t.Fatalf("failed to create memory: %v", err)
	}

	if mem.ID == 0 {
		t.Error("memory ID not set")
	}
}

func TestSearchMemories(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Create test memories
	mem1 := &models.Memory{Type: "semantic", Content: "Coffee preferences", Tags: "coffee", Importance: 5}
	mem2 := &models.Memory{Type: "semantic", Content: "Tea preferences", Tags: "tea", Importance: 5}
	mem3 := &models.Memory{Type: "episodic", Content: "Meeting notes", Tags: "work", Importance: 7}

	store.CreateMemory(mem1)
	store.CreateMemory(mem2)
	store.CreateMemory(mem3)

	tests := []struct {
		name     string
		query    string
		memType  string
		limit    int
		expected int
	}{
		{"search by content", "Coffee", "", 10, 1},
		{"search by tags", "coffee", "", 10, 1},
		{"filter by type", "", "semantic", 10, 2},
		{"combined search", "preferences", "semantic", 10, 2},
		{"limit results", "", "", 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memories, err := store.SearchMemories(tt.query, tt.memType, tt.limit)
			if err != nil {
				t.Fatalf("SearchMemories failed: %v", err)
			}
			if len(memories) != tt.expected {
				t.Errorf("expected %d memories, got %d", tt.expected, len(memories))
			}
		})
	}
}

func TestSearchMemoriesByTenant(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Create memories for different tenants
	mem1 := &models.Memory{
		Type:           "semantic",
		Content:        "User 1 likes coffee",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
	}
	mem2 := &models.Memory{
		Type:           "semantic",
		Content:        "User 2 likes tea",
		AppID:          "app1",
		ExternalUserID: "user2",
		Importance:     5,
	}
	mem3 := &models.Memory{
		Type:           "semantic",
		Content:        "User 1 likes chocolate",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
	}

	store.CreateMemory(mem1)
	store.CreateMemory(mem2)
	store.CreateMemory(mem3)

	memories, err := store.SearchMemoriesByTenant("app1", "user1", "likes", 10)
	if err != nil {
		t.Fatalf("SearchMemoriesByTenant failed: %v", err)
	}

	if len(memories) != 2 {
		t.Errorf("expected 2 memories for user1, got %d", len(memories))
	}

	// Verify tenant isolation
	memories2, _ := store.SearchMemoriesByTenant("app1", "user2", "likes", 10)
	if len(memories2) != 1 {
		t.Errorf("expected 1 memory for user2, got %d", len(memories2))
	}
}

func TestGetMemoryByID(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	mem := &models.Memory{
		Type:       "semantic",
		Content:    "Test memory",
		Importance: 5,
	}
	store.CreateMemory(mem)

	retrieved, err := store.GetMemoryByID(mem.ID)
	if err != nil {
		t.Fatalf("GetMemoryByID failed: %v", err)
	}

	if retrieved.Content != mem.Content {
		t.Errorf("expected content %s, got %s", mem.Content, retrieved.Content)
	}
}

func TestGetMemoryByIDAndTenant(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	mem := &models.Memory{
		Type:           "semantic",
		Content:        "Test memory",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
	}
	store.CreateMemory(mem)

	retrieved, err := store.GetMemoryByIDAndTenant(mem.ID, "app1", "user1")
	if err != nil {
		t.Fatalf("GetMemoryByIDAndTenant failed: %v", err)
	}

	if retrieved.Content != mem.Content {
		t.Errorf("expected content %s, got %s", mem.Content, retrieved.Content)
	}

	// Test tenant isolation - should not find memory with wrong tenant
	_, err = store.GetMemoryByIDAndTenant(mem.ID, "app1", "user2")
	if err == nil {
		t.Error("expected error when querying with wrong tenant")
	}
}

func TestDeleteMemory(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	mem := &models.Memory{
		Type:       "semantic",
		Content:    "Test memory",
		Importance: 5,
	}
	store.CreateMemory(mem)

	err := store.DeleteMemory(mem)
	if err != nil {
		t.Fatalf("DeleteMemory failed: %v", err)
	}

	// Verify deletion
	_, err = store.GetMemoryByID(mem.ID)
	if err == nil {
		t.Error("memory should be deleted")
	}
}

func TestEntityOperations(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Test CreateOrUpdateEntity
	ent := &models.Entity{
		Name: "user:test",
		Data: `{"key1":"value1"}`,
	}

	err := store.CreateOrUpdateEntity(ent)
	if err != nil {
		t.Fatalf("CreateOrUpdateEntity failed: %v", err)
	}

	if ent.ID == 0 {
		t.Error("entity ID not set")
	}

	// Test GetEntity
	retrieved, err := store.GetEntity("user:test")
	if err != nil {
		t.Fatalf("GetEntity failed: %v", err)
	}

	if retrieved.Name != "user:test" {
		t.Errorf("expected name %s, got %s", "user:test", retrieved.Name)
	}

	// Test update
	ent.Data = `{"key1":"value1","key2":"value2"}`
	err = store.CreateOrUpdateEntity(ent)
	if err != nil {
		t.Fatalf("UpdateEntity failed: %v", err)
	}

	retrieved, _ = store.GetEntity("user:test")
	if retrieved.Data != ent.Data {
		t.Errorf("entity not updated correctly")
	}

	// Test ListEntities
	entities, err := store.ListEntities()
	if err != nil {
		t.Fatalf("ListEntities failed: %v", err)
	}

	if len(entities) != 1 {
		t.Errorf("expected 1 entity, got %d", len(entities))
	}
}

func TestRelationOperations(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	rel := &models.Relation{
		From: "user:alice",
		To:   "user:bob",
		Type: "friend",
	}

	err := store.CreateOrUpdateRelation(rel)
	if err != nil {
		t.Fatalf("CreateOrUpdateRelation failed: %v", err)
	}

	if rel.ID == 0 {
		t.Error("relation ID not set")
	}

	// Test GetRelations
	relations, err := store.GetRelations("user:alice")
	if err != nil {
		t.Fatalf("GetRelations failed: %v", err)
	}

	if len(relations) != 1 {
		t.Errorf("expected 1 relation, got %d", len(relations))
	}

	if relations[0].Type != "friend" {
		t.Errorf("expected type 'friend', got %s", relations[0].Type)
	}
}

func TestGetStats(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Create test data
	store.CreateMemory(&models.Memory{Type: "semantic", Content: "Test", Importance: 5})
	store.CreateMemory(&models.Memory{Type: "semantic", Content: "Test2", Importance: 5})
	store.CreateOrUpdateEntity(&models.Entity{Name: "entity1", Data: "{}"})
	store.CreateOrUpdateRelation(&models.Relation{From: "a", To: "b", Type: "test"})

	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.Memories != 2 {
		t.Errorf("expected 2 memories, got %d", stats.Memories)
	}
	if stats.Entities != 1 {
		t.Errorf("expected 1 entity, got %d", stats.Entities)
	}
	if stats.Relations != 1 {
		t.Errorf("expected 1 relation, got %d", stats.Relations)
	}
}

func TestBundleOperations(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Create bundle
	bundle := &models.Bundle{
		Name:           "Test Bundle",
		AppID:          "app1",
		ExternalUserID: "user1",
	}

	err := store.CreateBundle(bundle)
	if err != nil {
		t.Fatalf("CreateBundle failed: %v", err)
	}

	if bundle.ID == 0 {
		t.Error("bundle ID not set")
	}

	// Test GetBundle
	retrieved, err := store.GetBundle(bundle.ID, "app1", "user1")
	if err != nil {
		t.Fatalf("GetBundle failed: %v", err)
	}

	if retrieved.Name != "Test Bundle" {
		t.Errorf("expected name %s, got %s", "Test Bundle", retrieved.Name)
	}

	// Test ListBundles
	bundles, err := store.ListBundles("app1", "user1")
	if err != nil {
		t.Fatalf("ListBundles failed: %v", err)
	}

	if len(bundles) != 1 {
		t.Errorf("expected 1 bundle, got %d", len(bundles))
	}

	// Test tenant isolation
	bundles2, _ := store.ListBundles("app1", "user2")
	if len(bundles2) != 0 {
		t.Errorf("expected 0 bundles for user2, got %d", len(bundles2))
	}

	// Test DeleteBundle
	err = store.DeleteBundle(bundle.ID, "app1", "user1")
	if err != nil {
		t.Fatalf("DeleteBundle failed: %v", err)
	}

	// Verify deletion
	_, err = store.GetBundle(bundle.ID, "app1", "user1")
	if err == nil {
		t.Error("bundle should be deleted")
	}
}

func TestSearchMemoriesByTenantAndBundle(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Create bundle
	bundle := &models.Bundle{
		Name:           "Coffee Bundle",
		AppID:          "app1",
		ExternalUserID: "user1",
	}
	store.CreateBundle(bundle)

	// Create memories with and without bundle
	mem1 := &models.Memory{
		Type:           "semantic",
		Content:        "Coffee preference",
		AppID:          "app1",
		ExternalUserID: "user1",
		BundleID:       &bundle.ID,
		Importance:     5,
	}
	mem2 := &models.Memory{
		Type:           "semantic",
		Content:        "Tea preference",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
	}
	mem3 := &models.Memory{
		Type:           "semantic",
		Content:        "Another coffee note",
		AppID:          "app1",
		ExternalUserID: "user1",
		BundleID:       &bundle.ID,
		Importance:     5,
	}

	store.CreateMemory(mem1)
	store.CreateMemory(mem2)
	store.CreateMemory(mem3)

	// Search without bundle filter
	memories, err := store.SearchMemoriesByTenantAndBundle("app1", "user1", "preference", nil, 10)
	if err != nil {
		t.Fatalf("SearchMemoriesByTenantAndBundle failed: %v", err)
	}
	if len(memories) != 2 {
		t.Errorf("expected 2 memories, got %d", len(memories))
	}

	// Search with bundle filter
	bundleMemories, err := store.SearchMemoriesByTenantAndBundle("app1", "user1", "coffee", &bundle.ID, 10)
	if err != nil {
		t.Fatalf("SearchMemoriesByTenantAndBundle with bundle failed: %v", err)
	}
	if len(bundleMemories) != 2 {
		t.Errorf("expected 2 memories in bundle, got %d", len(bundleMemories))
	}
}
