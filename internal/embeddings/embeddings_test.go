package embeddings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{0.0, 1.0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{-1.0, 0.0},
			expected: -1.0,
		},
		{
			name:     "different length",
			a:        []float32{1.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.a, tt.b)
			if result < tt.expected-0.01 || result > tt.expected+0.01 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestEncodeDecodeVector(t *testing.T) {
	original := []float32{0.1, 0.2, 0.3, 0.4, 0.5}

	encoded, err := EncodeVector(original)
	if err != nil {
		t.Fatalf("failed to encode vector: %v", err)
	}

	decoded, err := DecodeVector(encoded)
	if err != nil {
		t.Fatalf("failed to decode vector: %v", err)
	}

	if len(decoded) != len(original) {
		t.Fatalf("length mismatch: expected %d, got %d", len(original), len(decoded))
	}

	for i := range original {
		if decoded[i] != original[i] {
			t.Errorf("value mismatch at index %d: expected %f, got %f", i, original[i], decoded[i])
		}
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		metadata map[string]any
		expected string
	}{
		{
			name:     "explicit content type in metadata",
			content:  "some content",
			metadata: map[string]any{"contentType": "image/png"},
			expected: "image/png",
		},
		{
			name:     "base64 image",
			content:  "data:image/jpeg;base64,/9j/4AAQSkZJRg==",
			metadata: map[string]any{},
			expected: "image/jpeg",
		},
		{
			name:     "image URL",
			content:  "https://example.com/image.jpg",
			metadata: map[string]any{},
			expected: "image/jpeg",
		},
		{
			name:     "PDF URL",
			content:  "https://example.com/document.pdf",
			metadata: map[string]any{},
			expected: "application/pdf",
		},
		{
			name:     "default text",
			content:  "plain text content",
			metadata: map[string]any{},
			expected: "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectContentType(tt.content, tt.metadata)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestLocalEmbeddingService(t *testing.T) {
	service := NewLocalEmbeddingService()

	embedding, err := service.GenerateEmbedding("test content", "text/plain")
	if err != nil {
		t.Fatalf("failed to generate embedding: %v", err)
	}

	if len(embedding) == 0 {
		t.Error("embedding is empty")
	}

	if len(embedding) != service.dimension {
		t.Errorf("expected dimension %d, got %d", service.dimension, len(embedding))
	}

	// Teste dass ähnliche Inhalte ähnliche Embeddings haben
	embedding2, err := service.GenerateEmbedding("test content", "text/plain")
	if err != nil {
		t.Fatalf("failed to generate second embedding: %v", err)
	}

	similarity := CosineSimilarity(embedding, embedding2)
	if similarity < 0.99 {
		t.Errorf("identical content should have high similarity, got %f", similarity)
	}

	// Teste dass verschiedene Inhalte unterschiedliche Embeddings haben
	embedding3, err := service.GenerateEmbedding("completely different content", "text/plain")
	if err != nil {
		t.Fatalf("failed to generate third embedding: %v", err)
	}

	similarity2 := CosineSimilarity(embedding, embedding3)
	if similarity2 > 0.9 {
		t.Errorf("different content should have lower similarity, got %f", similarity2)
	}
}

// TestEmbeddingServiceInterface testet dass beide Services das Interface korrekt implementieren
func TestEmbeddingServiceInterface(t *testing.T) {
	services := []struct {
		name    string
		service EmbeddingService
	}{
		{"LocalEmbeddingService", NewLocalEmbeddingService()},
	}

	for _, tt := range services {
		t.Run(tt.name, func(t *testing.T) {
			// Teste GenerateEmbedding
			embedding, err := tt.service.GenerateEmbedding("test content", "text/plain")
			if err != nil {
				t.Fatalf("GenerateEmbedding failed: %v", err)
			}
			if len(embedding) == 0 {
				t.Error("embedding is empty")
			}
			if len(embedding) != 384 {
				t.Errorf("expected dimension 384, got %d", len(embedding))
			}

			// Teste GenerateEmbeddingsBatch
			embeddings, err := tt.service.GenerateEmbeddingsBatch([]string{"test1", "test2"}, "text/plain")
			if err != nil {
				t.Fatalf("GenerateEmbeddingsBatch failed: %v", err)
			}
			if len(embeddings) != 2 {
				t.Errorf("expected 2 embeddings, got %d", len(embeddings))
			}
			for i, emb := range embeddings {
				if len(emb) != 384 {
					t.Errorf("embedding %d: expected dimension 384, got %d", i, len(emb))
				}
			}
		})
	}
}

// TestGetEmbeddingService testet die Fallback-Logik von GetEmbeddingService
func TestGetEmbeddingService(t *testing.T) {
	// Reset vor jedem Test
	resetEmbeddingService()

	t.Run("without CORTEX_EMBEDDING_MODEL_PATH", func(t *testing.T) {
		resetEmbeddingService()
		oldPath := os.Getenv("CORTEX_EMBEDDING_MODEL_PATH")
		defer os.Setenv("CORTEX_EMBEDDING_MODEL_PATH", oldPath)
		os.Unsetenv("CORTEX_EMBEDDING_MODEL_PATH")

		service := GetEmbeddingService()
		if service == nil {
			t.Fatal("GetEmbeddingService returned nil")
		}

		// Sollte LocalEmbeddingService sein
		_, ok := service.(*LocalEmbeddingService)
		if !ok {
			t.Error("expected LocalEmbeddingService when CORTEX_EMBEDDING_MODEL_PATH is not set")
		}

		// Teste dass es funktioniert
		embedding, err := service.GenerateEmbedding("test", "text/plain")
		if err != nil {
			t.Fatalf("GenerateEmbedding failed: %v", err)
		}
		if len(embedding) != 384 {
			t.Errorf("expected dimension 384, got %d", len(embedding))
		}
	})

	t.Run("with CORTEX_EMBEDDING_MODEL_PATH but model missing", func(t *testing.T) {
		resetEmbeddingService()
		oldPath := os.Getenv("CORTEX_EMBEDDING_MODEL_PATH")
		defer os.Setenv("CORTEX_EMBEDDING_MODEL_PATH", oldPath)
		os.Setenv("CORTEX_EMBEDDING_MODEL_PATH", "/nonexistent/path/model.gtemodel")

		service := GetEmbeddingService()
		if service == nil {
			t.Fatal("GetEmbeddingService returned nil")
		}

		// Sollte auf LocalEmbeddingService zurückfallen
		_, ok := service.(*LocalEmbeddingService)
		if !ok {
			t.Error("expected LocalEmbeddingService fallback when model is missing")
		}
	})

	t.Run("with CORTEX_EMBEDDING_MODEL_PATH and valid model", func(t *testing.T) {
		resetEmbeddingService()
		oldPath := os.Getenv("CORTEX_EMBEDDING_MODEL_PATH")
		defer os.Setenv("CORTEX_EMBEDDING_MODEL_PATH", oldPath)

		// Prüfe ob Modell existiert
		homeDir, _ := os.UserHomeDir()
		modelPath := filepath.Join(homeDir, ".openclaw", "gte-small.gtemodel")
		if _, err := os.Stat(modelPath); os.IsNotExist(err) {
			t.Skip("GTE model not found, skipping test")
		}

		os.Setenv("CORTEX_EMBEDDING_MODEL_PATH", modelPath)
		service := GetEmbeddingService()
		if service == nil {
			t.Fatal("GetEmbeddingService returned nil")
		}

		// Sollte GTEEmbeddingService sein
		gteService, ok := service.(*GTEEmbeddingService)
		if !ok {
			t.Error("expected GTEEmbeddingService when model is available")
		}

		// Teste dass es funktioniert
		embedding, err := service.GenerateEmbedding("test content", "text/plain")
		if err != nil {
			t.Fatalf("GenerateEmbedding failed: %v", err)
		}
		if len(embedding) != 384 {
			t.Errorf("expected dimension 384, got %d", len(embedding))
		}

		// Cleanup
		if gteService != nil {
			gteService.Close()
		}
	})
}

// TestGTEEmbeddingService testet den GTE-Service direkt (optional, skip wenn Modell fehlt)
func TestGTEEmbeddingService(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	modelPath := filepath.Join(homeDir, ".openclaw", "gte-small.gtemodel")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skip("GTE model not found, skipping test")
	}

	service, err := NewGTEEmbeddingService(modelPath)
	if err != nil {
		t.Fatalf("Failed to create GTEEmbeddingService: %v", err)
	}
	defer service.Close()

	// Test GenerateEmbedding
	embedding, err := service.GenerateEmbedding("test content", "text/plain")
	if err != nil {
		t.Fatalf("GenerateEmbedding failed: %v", err)
	}
	if len(embedding) != 384 {
		t.Errorf("expected dimension 384, got %d", len(embedding))
	}

	// Test dass identische Inhalte ähnliche Embeddings haben
	embedding2, err := service.GenerateEmbedding("test content", "text/plain")
	if err != nil {
		t.Fatalf("GenerateEmbedding failed: %v", err)
	}
	similarity := CosineSimilarity(embedding, embedding2)
	if similarity < 0.99 {
		t.Errorf("identical content should have high similarity, got %f", similarity)
	}

	// Test GenerateEmbeddingsBatch
	embeddings, err := service.GenerateEmbeddingsBatch([]string{"test1", "test2", "test3"}, "text/plain")
	if err != nil {
		t.Fatalf("GenerateEmbeddingsBatch failed: %v", err)
	}
	if len(embeddings) != 3 {
		t.Errorf("expected 3 embeddings, got %d", len(embeddings))
	}
	for i, emb := range embeddings {
		if len(emb) != 384 {
			t.Errorf("embedding %d: expected dimension 384, got %d", i, len(emb))
		}
	}

	// Test Close
	err = service.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
