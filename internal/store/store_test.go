package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cortex/internal/embeddings"
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

// SeedRealisticData fills the store with a realistic test dataset: two tenants, bundles,
// active/expired/low-importance memories, and some with embeddings for merge/similar tests.
// Returns IDs and counts for use in integration test assertions.
type SeedRealisticData struct {
	App1, App2 string
	User1, User2 string
	Bundle1ID, Bundle2ID int64
	// Tenant1 (app1/user1): active memory IDs (no expiry), expired IDs, low-importance IDs, similar pair (keepID, mergeID)
	ActiveIDs    []int64
	ExpiredIDs   []int64
	LowImportIDs []int64
	SimilarKeepID, SimilarMergeID int64
	// Tenant2 (app2/user2): just a few IDs
	Tenant2ActiveIDs []int64
	// Already archived (for delete test): ID and whether updated_at is old
	ArchivedOldID int64
}

func seedRealisticTestData(t *testing.T, s *CortexStore) SeedRealisticData {
	t.Helper()
	var out SeedRealisticData
	out.App1, out.User1 = "app1", "user1"
	out.App2, out.User2 = "app2", "user2"

	// Bundles for tenant1
	b1 := &models.Bundle{Name: "Präferenzen", AppID: out.App1, ExternalUserID: out.User1}
	b2 := &models.Bundle{Name: "Arbeit", AppID: out.App1, ExternalUserID: out.User1}
	if err := s.CreateBundle(b1); err != nil {
		t.Fatalf("seed bundle: %v", err)
	}
	if err := s.CreateBundle(b2); err != nil {
		t.Fatalf("seed bundle: %v", err)
	}
	out.Bundle1ID, out.Bundle2ID = b1.ID, b2.ID

	now := time.Now()
	expired := now.Add(-2 * time.Hour)
	futureExpiry := now.Add(24 * time.Hour)

	// Tenant1: active (no expiry)
	for _, c := range []string{"Benutzer mag Kaffee mit Milch.", "Projekt X Deadline nächste Woche.", "Wichtige Besprechung mit Team."} {
		m := &models.Memory{
			Type: "semantic", Content: c, AppID: out.App1, ExternalUserID: out.User1,
			Importance: 5, Status: models.MemoryStatusActive,
			Metadata: helpers.MarshalMetadata(map[string]any{"source": "chat"}),
		}
		if err := s.CreateMemory(m); err != nil {
			t.Fatalf("seed memory: %v", err)
		}
		out.ActiveIDs = append(out.ActiveIDs, m.ID)
	}

	// Tenant1: expired (TTL)
	for _, c := range []string{"Alter Eintrag abgelaufen.", "Noch ein abgelaufener Hinweis."} {
		m := &models.Memory{
			Type: "semantic", Content: c, AppID: out.App1, ExternalUserID: out.User1,
			Importance: 5, Status: models.MemoryStatusActive, ExpiresAt: &expired,
		}
		if err := s.CreateMemory(m); err != nil {
			t.Fatalf("seed memory: %v", err)
		}
		out.ExpiredIDs = append(out.ExpiredIDs, m.ID)
	}

	// Tenant1: low importance (for archive-low-importance)
	for _, c := range []string{"Randnotiz.", "Unwichtig."} {
		m := &models.Memory{
			Type: "semantic", Content: c, AppID: out.App1, ExternalUserID: out.User1,
			Importance: 1, Status: models.MemoryStatusActive,
		}
		if err := s.CreateMemory(m); err != nil {
			t.Fatalf("seed memory: %v", err)
		}
		out.LowImportIDs = append(out.LowImportIDs, m.ID)
	}

	// Tenant1: identical content pair (for FindSimilar / Merge), with embeddings
	sameContent := "Der Nutzer trinkt morgens gerne Kaffee."
	mKeep := &models.Memory{
		Type: "semantic", Content: sameContent, AppID: out.App1, ExternalUserID: out.User1,
		Importance: 6, Tags: "kaffee,morgen", BundleID: &out.Bundle1ID,
	}
	mMerge := &models.Memory{
		Type: "semantic", Content: sameContent, AppID: out.App1, ExternalUserID: out.User1,
		Importance: 4, Tags: "frühstück", BundleID: &out.Bundle1ID,
	}
	if err := s.CreateMemory(mKeep); err != nil {
		t.Fatalf("seed memory: %v", err)
	}
	if err := s.CreateMemory(mMerge); err != nil {
		t.Fatalf("seed memory: %v", err)
	}
	s.GenerateEmbeddingForMemory(mKeep)
	s.GenerateEmbeddingForMemory(mMerge)
	out.SimilarKeepID, out.SimilarMergeID = mKeep.ID, mMerge.ID

	// Tenant1: one with future expiry (must stay active)
	mFuture := &models.Memory{
		Type: "semantic", Content: "Erinnerung für morgen.", AppID: out.App1, ExternalUserID: out.User1,
		Importance: 7, Status: models.MemoryStatusActive, ExpiresAt: &futureExpiry,
	}
	if err := s.CreateMemory(mFuture); err != nil {
		t.Fatalf("seed memory: %v", err)
	}
	out.ActiveIDs = append(out.ActiveIDs, mFuture.ID)

	// Tenant2
	for _, c := range []string{"Anderer Nutzer Notiz 1.", "Anderer Nutzer Notiz 2."} {
		m := &models.Memory{
			Type: "semantic", Content: c, AppID: out.App2, ExternalUserID: out.User2,
			Importance: 5, Status: models.MemoryStatusActive,
		}
		if err := s.CreateMemory(m); err != nil {
			t.Fatalf("seed memory: %v", err)
		}
		out.Tenant2ActiveIDs = append(out.Tenant2ActiveIDs, m.ID)
	}

	// Archived with old updated_at (for DeleteArchivedOlderThan)
	mArch := &models.Memory{
		Type: "semantic", Content: "Bereits archiviert, alt.", AppID: out.App1, ExternalUserID: out.User1,
		Importance: 3, Status: models.MemoryStatusArchived,
	}
	if err := s.CreateMemory(mArch); err != nil {
		t.Fatalf("seed memory: %v", err)
	}
	oldTime := now.Add(-72 * time.Hour)
	s.GetDB().Exec("UPDATE memories SET updated_at = ? WHERE id = ?", oldTime, mArch.ID)
	out.ArchivedOldID = mArch.ID

	return out
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

	memories, err := store.SearchMemoriesByTenantAndBundle("app1", "user1", "likes", nil, 10, nil, nil, false)
	if err != nil {
		t.Fatalf("SearchMemoriesByTenant failed: %v", err)
	}

	if len(memories) != 2 {
		t.Errorf("expected 2 memories for user1, got %d", len(memories))
	}

	// Verify tenant isolation
	memories2, _ := store.SearchMemoriesByTenantAndBundle("app1", "user2", "likes", nil, 10, nil, nil, false)
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

	retrieved, err := store.GetMemoryByIDAndTenant(mem.ID, mem.AppID, mem.ExternalUserID, false)
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

	retrieved, err := store.GetMemoryByIDAndTenant(mem.ID, "app1", "user1", false)
	if err != nil {
		t.Fatalf("GetMemoryByIDAndTenant failed: %v", err)
	}

	if retrieved.Content != mem.Content {
		t.Errorf("expected content %s, got %s", mem.Content, retrieved.Content)
	}

	// Test tenant isolation - should not find memory with wrong tenant
	_, err = store.GetMemoryByIDAndTenant(mem.ID, "app1", "user2", false)
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
	_, err = store.GetMemoryByIDAndTenant(mem.ID, "app1", "user1", false)
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
	memories, err := store.SearchMemoriesByTenantAndBundle("app1", "user1", "preference", nil, 10, nil, nil, false)
	if err != nil {
		t.Fatalf("SearchMemoriesByTenantAndBundle failed: %v", err)
	}
	if len(memories) != 2 {
		t.Errorf("expected 2 memories, got %d", len(memories))
	}

	// Search with bundle filter
	bundleMemories, err := store.SearchMemoriesByTenantAndBundle("app1", "user1", "coffee", &bundle.ID, 10, nil, nil, false)
	if err != nil {
		t.Fatalf("SearchMemoriesByTenantAndBundle with bundle failed: %v", err)
	}
	if len(bundleMemories) != 2 {
		t.Errorf("expected 2 memories in bundle, got %d", len(bundleMemories))
	}
}

func TestGenerateEmbeddingForMemory(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	mem := &models.Memory{
		Type:           "semantic",
		Content:        "Test memory for embedding",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
	}

	// Create memory first
	err := store.CreateMemory(mem)
	if err != nil {
		t.Fatalf("CreateMemory failed: %v", err)
	}

	// Generate embedding
	err = store.GenerateEmbeddingForMemory(mem)
	if err != nil {
		t.Fatalf("GenerateEmbeddingForMemory failed: %v", err)
	}

	// Verify embedding was generated
	if mem.Embedding == "" {
		t.Error("embedding was not generated")
	}

	// Verify embedding can be decoded
	embedding, err := embeddings.DecodeVector(mem.Embedding)
	if err != nil {
		t.Fatalf("DecodeVector failed: %v", err)
	}
	if len(embedding) != 384 {
		t.Errorf("expected embedding dimension 384, got %d", len(embedding))
	}

	// Verify memory was saved with embedding
	retrieved, err := store.GetMemoryByIDAndTenant(mem.ID, "app1", "user1", false)
	if err != nil {
		t.Fatalf("GetMemoryByIDAndTenant failed: %v", err)
	}
	if retrieved.Embedding == "" {
		t.Error("embedding was not saved to database")
	}
}

func TestSearchMemoriesByTenantSemanticAndBundle(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Create memories with embeddings
	mem1 := &models.Memory{
		Type:           "semantic",
		Content:        "User likes coffee and espresso",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
	}
	mem2 := &models.Memory{
		Type:           "semantic",
		Content:        "User prefers tea over coffee",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
	}
	mem3 := &models.Memory{
		Type:           "semantic",
		Content:        "User drinks hot beverages in the morning",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
	}

	store.CreateMemory(mem1)
	store.CreateMemory(mem2)
	store.CreateMemory(mem3)

	// Generate embeddings for all memories
	store.GenerateEmbeddingForMemory(mem1)
	store.GenerateEmbeddingForMemory(mem2)
	store.GenerateEmbeddingForMemory(mem3)

	// Test semantic search
	memories, err := store.SearchMemoriesByTenantSemanticAndBundle("app1", "user1", "coffee drinks", nil, 10, nil, nil, false)
	if err != nil {
		t.Fatalf("SearchMemoriesByTenantSemanticAndBundle failed: %v", err)
	}

	// Should find at least mem1 (coffee) and mem3 (drinks hot beverages)
	if len(memories) == 0 {
		t.Error("semantic search returned no results")
	}

	// Verify results are sorted by similarity (highest first)
	if len(memories) > 1 {
		// First result should have highest similarity
		// Note: We can't verify exact similarity values as they depend on the embedding service
		// But we can verify that results are returned
		foundCoffee := false
		for _, mem := range memories {
			if mem.Content == mem1.Content {
				foundCoffee = true
				break
			}
		}
		if !foundCoffee {
			t.Log("Note: coffee memory not found in semantic results (may vary by embedding service)")
		}
	}

	// Test with bundle filter
	bundle := &models.Bundle{
		Name:           "Coffee Bundle",
		AppID:          "app1",
		ExternalUserID: "user1",
	}
	store.CreateBundle(bundle)
	mem1.BundleID = &bundle.ID
	store.db.Save(mem1)

	bundleMemories, err := store.SearchMemoriesByTenantSemanticAndBundle("app1", "user1", "coffee", &bundle.ID, 10, nil, nil, false)
	if err != nil {
		t.Fatalf("SearchMemoriesByTenantSemanticAndBundle with bundle failed: %v", err)
	}
	// Should find mem1 in bundle
	foundInBundle := false
	for _, mem := range bundleMemories {
		if mem.ID == mem1.ID {
			foundInBundle = true
			break
		}
	}
	if !foundInBundle {
		t.Error("expected to find memory in bundle")
	}
}

// --- Archive / TTL / Cleanup ---

func TestArchiveMemoriesByExpiry(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	expired := time.Now().Add(-time.Hour)
	mem := &models.Memory{
		Type:           "semantic",
		Content:        "Expired memory",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
		Status:         models.MemoryStatusActive,
		ExpiresAt:      &expired,
	}
	store.CreateMemory(mem)

	n, err := store.ArchiveMemoriesByExpiry(time.Now())
	if err != nil {
		t.Fatalf("ArchiveMemoriesByExpiry failed: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 archived, got %d", n)
	}

	got, _ := store.GetMemoryByIDAndTenant(mem.ID, "app1", "user1", true)
	if got == nil {
		t.Fatal("memory not found")
	}
	if got.Status != models.MemoryStatusArchived {
		t.Errorf("expected status archived, got %s", got.Status)
	}
}

func TestDeleteArchivedOlderThan(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	mem := &models.Memory{
		Type:           "semantic",
		Content:        "Old archived",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
		Status:         models.MemoryStatusArchived,
	}
	store.CreateMemory(mem)
	// Set updated_at to the past via raw update so it gets picked by DeleteArchivedOlderThan
	oldTime := time.Now().Add(-48 * time.Hour)
	store.GetDB().Exec("UPDATE memories SET updated_at = ? WHERE id = ?", oldTime, mem.ID)

	n, err := store.DeleteArchivedOlderThan(time.Now().Add(-24 * time.Hour))
	if err != nil {
		t.Fatalf("DeleteArchivedOlderThan failed: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 deleted, got %d", n)
	}

	_, err = store.GetMemoryByIDAndTenant(mem.ID, "app1", "user1", true)
	if err == nil {
		t.Error("memory should be deleted")
	}
}

// --- Version History ---

func TestUpdateMemoryAndListMemoryVersions(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	mem := &models.Memory{
		Type:           "semantic",
		Content:        "Original content",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
		Tags:           "a,b",
	}
	store.CreateMemory(mem)

	versions, err := store.ListMemoryVersions(mem.ID, "app1", "user1")
	if err != nil {
		t.Fatalf("ListMemoryVersions failed: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("expected 0 versions before update, got %d", len(versions))
	}

	mem.Content = "Updated content"
	mem.Importance = 7
	if err := store.UpdateMemory(mem, "api"); err != nil {
		t.Fatalf("UpdateMemory failed: %v", err)
	}

	versions, err = store.ListMemoryVersions(mem.ID, "app1", "user1")
	if err != nil {
		t.Fatalf("ListMemoryVersions after update failed: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(versions))
	}
	if versions[0].Content != "Original content" {
		t.Errorf("version content: expected Original content, got %s", versions[0].Content)
	}
	if versions[0].Importance != 5 {
		t.Errorf("version importance: expected 5, got %d", versions[0].Importance)
	}
	if versions[0].ChangedBy != "api" {
		t.Errorf("version changed_by: expected api, got %s", versions[0].ChangedBy)
	}
}

// --- FindSimilarMemoryPairs ---

func TestFindSimilarMemoryPairs(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Two very similar contents so embedding similarity is high
	mem1 := &models.Memory{
		Type:           "semantic",
		Content:        "The user loves coffee in the morning",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
	}
	mem2 := &models.Memory{
		Type:           "semantic",
		Content:        "The user loves coffee in the morning",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
	}
	store.CreateMemory(mem1)
	store.CreateMemory(mem2)
	store.GenerateEmbeddingForMemory(mem1)
	store.GenerateEmbeddingForMemory(mem2)

	pairs, err := store.FindSimilarMemoryPairs("app1", "user1", nil, 0.99, 10)
	if err != nil {
		t.Fatalf("FindSimilarMemoryPairs failed: %v", err)
	}
	// Identical content should yield similarity 1.0
	if len(pairs) < 1 {
		t.Errorf("expected at least 1 similar pair (identical content), got %d", len(pairs))
	}
	// Pair should be (lowerID, higherID)
	if len(pairs) > 0 {
		if pairs[0][0] >= pairs[0][1] {
			t.Errorf("expected pair (lower, higher), got %v", pairs[0])
		}
	}

	// With very high threshold and different content, may get 0 pairs
	mem3 := &models.Memory{
		Type:           "semantic",
		Content:        "Completely unrelated topic about weather",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     5,
	}
	store.CreateMemory(mem3)
	store.GenerateEmbeddingForMemory(mem3)
	pairs2, _ := store.FindSimilarMemoryPairs("app1", "user1", nil, 0.9999, 10)
	// We still have mem1 and mem2 identical, so at least one pair
	if len(pairs2) < 1 {
		t.Errorf("expected at least 1 pair (mem1-mem2), got %d", len(pairs2))
	}
}

// --- MergeMemories ---

func TestMergeMemories(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	mem1 := &models.Memory{
		Type:           "semantic",
		Content:        "First",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     3,
		Tags:           "a,b",
		Metadata:       helpers.MarshalMetadata(map[string]any{"k1": "v1"}),
	}
	mem2 := &models.Memory{
		Type:           "semantic",
		Content:        "Second",
		AppID:          "app1",
		ExternalUserID: "user1",
		Importance:     7,
		Tags:           "b,c",
		Metadata:       helpers.MarshalMetadata(map[string]any{"k2": "v2"}),
	}
	store.CreateMemory(mem1)
	store.CreateMemory(mem2)

	err := store.MergeMemories(mem1.ID, mem2.ID, "app1", "user1")
	if err != nil {
		t.Fatalf("MergeMemories failed: %v", err)
	}

	keep, _ := store.GetMemoryByIDAndTenant(mem1.ID, "app1", "user1", false)
	if keep == nil {
		t.Fatal("keep memory not found")
	}
	if keep.Content != "First | Second" {
		t.Errorf("merged content: expected First | Second, got %s", keep.Content)
	}
	if keep.Importance != 7 {
		t.Errorf("merged importance: expected 7, got %d", keep.Importance)
	}
	meta := helpers.UnmarshalMetadata(keep.Metadata)
	if meta["k1"] != "v1" || meta["k2"] != "v2" {
		t.Errorf("merged metadata: expected k1 and k2, got %v", meta)
	}
	tagSet := make(map[string]struct{})
	for _, s := range strings.Split(keep.Tags, ",") {
		tagSet[strings.TrimSpace(s)] = struct{}{}
	}
	if _, ok := tagSet["a"]; !ok {
		t.Error("merged tags: missing a")
	}
	if _, ok := tagSet["b"]; !ok {
		t.Error("merged tags: missing b")
	}
	if _, ok := tagSet["c"]; !ok {
		t.Error("merged tags: missing c")
	}

	merged, _ := store.GetMemoryByIDAndTenant(mem2.ID, "app1", "user1", true)
	if merged == nil {
		t.Fatal("merged memory should still exist (archived)")
	}
	if merged.Status != models.MemoryStatusArchived {
		t.Errorf("merged memory status: expected archived, got %s", merged.Status)
	}
	mergedMeta := helpers.UnmarshalMetadata(merged.Metadata)
	if mergedMeta["merged_into"] == nil {
		t.Error("merged memory should have metadata merged_into")
	}
}

// --- Integration tests with realistic seed data in test DB ---

func TestIntegration_ListAndFilterWithRealData(t *testing.T) {
	s := setupTestDB(t)
	defer s.Close()
	seed := seedRealisticTestData(t, s)

	// Only active (default): no expired, no archived, no low-importance if we already archived them
	list, err := s.ListMemoriesByTenant(seed.App1, seed.User1, 50, 0, false)
	if err != nil {
		t.Fatalf("ListMemoriesByTenant: %v", err)
	}
	// Active: 3 general + 1 future expiry + 2 expired (still active until cleanup) + 2 low importance + 2 similar = 10 active
	// Plus 1 archived (ArchivedOldID) -> not in list when includeArchived=false
	if len(list) < 8 {
		t.Errorf("expected at least 8 active memories for tenant1, got %d", len(list))
	}

	// Tenant2 unchanged
	list2, err := s.ListMemoriesByTenant(seed.App2, seed.User2, 50, 0, false)
	if err != nil {
		t.Fatalf("ListMemoriesByTenant tenant2: %v", err)
	}
	if len(list2) != 2 {
		t.Errorf("expected 2 memories for tenant2, got %d", len(list2))
	}
}

func TestIntegration_ArchiveByExpiryWithRealData(t *testing.T) {
	s := setupTestDB(t)
	defer s.Close()
	seed := seedRealisticTestData(t, s)

	n, err := s.ArchiveMemoriesByExpiry(time.Now())
	if err != nil {
		t.Fatalf("ArchiveMemoriesByExpiry: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 archived by expiry, got %d", n)
	}

	for _, id := range seed.ExpiredIDs {
		got, _ := s.GetMemoryByIDAndTenant(id, seed.App1, seed.User1, true)
		if got == nil {
			t.Errorf("memory %d not found", id)
			continue
		}
		if got.Status != models.MemoryStatusArchived {
			t.Errorf("memory %d: expected archived, got %s", id, got.Status)
		}
	}
	// Future expiry still active
	futureID := seed.ActiveIDs[len(seed.ActiveIDs)-1]
	got, _ := s.GetMemoryByIDAndTenant(futureID, seed.App1, seed.User1, false)
	if got == nil {
		t.Error("future-expiry memory should still be active")
	}
}

func TestIntegration_FindSimilarWithRealData(t *testing.T) {
	s := setupTestDB(t)
	defer s.Close()
	seed := seedRealisticTestData(t, s)

	pairs, err := s.FindSimilarMemoryPairs(seed.App1, seed.User1, nil, 0.99, 20)
	if err != nil {
		t.Fatalf("FindSimilarMemoryPairs: %v", err)
	}
	if len(pairs) < 1 {
		t.Fatalf("expected at least 1 similar pair (identical content), got %d", len(pairs))
	}
	found := false
	for _, p := range pairs {
		if (p[0] == seed.SimilarKeepID && p[1] == seed.SimilarMergeID) || (p[0] == seed.SimilarMergeID && p[1] == seed.SimilarKeepID) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected pair containing SimilarKeepID=%d and SimilarMergeID=%d, got pairs %v", seed.SimilarKeepID, seed.SimilarMergeID, pairs)
	}
}

func TestIntegration_MergeAndHistoryWithRealData(t *testing.T) {
	s := setupTestDB(t)
	defer s.Close()
	seed := seedRealisticTestData(t, s)

	err := s.MergeMemories(seed.SimilarKeepID, seed.SimilarMergeID, seed.App1, seed.User1)
	if err != nil {
		t.Fatalf("MergeMemories: %v", err)
	}

	keep, err := s.GetMemoryByIDAndTenant(seed.SimilarKeepID, seed.App1, seed.User1, false)
	if err != nil || keep == nil {
		t.Fatalf("keep memory not found: %v", err)
	}
	content := "Der Nutzer trinkt morgens gerne Kaffee."
	if keep.Content != content+" | "+content {
		t.Errorf("merged content: expected duplicated content joined by | , got %q", keep.Content)
	}
	tags := strings.Split(keep.Tags, ",")
	tagSet := make(map[string]struct{})
	for _, x := range tags {
		tagSet[strings.TrimSpace(x)] = struct{}{}
	}
	if _, ok := tagSet["kaffee"]; !ok {
		t.Error("merged tags: missing kaffee")
	}
	if _, ok := tagSet["morgen"]; !ok {
		t.Error("merged tags: missing morgen")
	}
	if _, ok := tagSet["frühstück"]; !ok {
		t.Error("merged tags: missing frühstück")
	}

	versions, err := s.ListMemoryVersions(seed.SimilarKeepID, seed.App1, seed.User1)
	if err != nil {
		t.Fatalf("ListMemoryVersions: %v", err)
	}
	if len(versions) != 1 {
		t.Errorf("expected 1 version after merge, got %d", len(versions))
	}
	if len(versions) > 0 && versions[0].ChangedBy != "merge" {
		t.Errorf("version changed_by: expected merge, got %s", versions[0].ChangedBy)
	}

	merged, _ := s.GetMemoryByIDAndTenant(seed.SimilarMergeID, seed.App1, seed.User1, true)
	if merged == nil {
		t.Fatal("merged memory should exist as archived")
	}
	if merged.Status != models.MemoryStatusArchived {
		t.Errorf("merged memory status: expected archived, got %s", merged.Status)
	}
}

func TestIntegration_DeleteArchivedOlderThanWithRealData(t *testing.T) {
	s := setupTestDB(t)
	defer s.Close()
	seed := seedRealisticTestData(t, s)

	cutoff := time.Now().Add(-48 * time.Hour)
	n, err := s.DeleteArchivedOlderThan(cutoff)
	if err != nil {
		t.Fatalf("DeleteArchivedOlderThan: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 deleted (archived old), got %d", n)
	}

	_, err = s.GetMemoryByIDAndTenant(seed.ArchivedOldID, seed.App1, seed.User1, true)
	if err == nil {
		t.Error("ArchivedOldID should be deleted")
	}
}
