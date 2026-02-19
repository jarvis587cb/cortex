package embeddings

import (
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
}
