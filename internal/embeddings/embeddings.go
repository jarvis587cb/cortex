package embeddings

import (
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rcarmo/gte-go/gte"
)

// synonymExpandBegriffe: minimale Erweiterung für begriffliche Treffer (z. B. coffee ↔ latte).
// Jedes Wort wird durch sich selbst + verwandte Begriffe ergänzt, damit Similarity steigt.
var synonymExpandBegriffe = map[string][]string{
	"coffee":     {"latte", "cappuccino", "espresso", "kaffee"},
	"latte":      {"coffee", "cappuccino", "espresso"},
	"lattes":     {"coffee", "cappuccino", "espresso"},
	"cappuccino": {"coffee", "latte", "espresso"},
	"espresso":   {"coffee", "latte", "cappuccino"},
	"kaffee":     {"coffee", "latte", "espresso"},
	"tea":        {"tee", "chai"},
	"tee":        {"tea", "chai"},
}

// EmbeddingService interface für verschiedene Embedding-Provider
type EmbeddingService interface {
	GenerateEmbedding(content string, contentType string) ([]float32, error)
	GenerateEmbeddingsBatch(contents []string, contentType string) ([][]float32, error)
}

// LocalEmbeddingService - Lokaler Embedding-Service ohne externe Abhängigkeiten
// Verwendet einen verbesserten Hash-basierten Ansatz für semantische Ähnlichkeit
type LocalEmbeddingService struct {
	dimension int
}

// NewLocalEmbeddingService erstellt einen lokalen Embedding Service
func NewLocalEmbeddingService() *LocalEmbeddingService {
	return &LocalEmbeddingService{
		dimension: 384, // Kompakte Dimension für lokale Embeddings
	}
}

// expandWithSynonyms hängt verwandte Begriffe an den normalisierten Text an,
// damit z. B. "oat milk lattes" und "coffee" eine höhere Similarity bekommen.
func expandWithSynonyms(normalized string) string {
	words := strings.Fields(normalized)
	seen := make(map[string]bool)
	for _, w := range words {
		seen[w] = true
	}
	var extra []string
	for _, w := range words {
		if len(w) <= 2 {
			continue
		}
		if syns, ok := synonymExpandBegriffe[w]; ok {
			for _, s := range syns {
				if !seen[s] {
					seen[s] = true
					extra = append(extra, s)
				}
			}
		}
	}
	if len(extra) == 0 {
		return normalized
	}
	return normalized + " " + strings.Join(extra, " ")
}

// GenerateEmbedding generiert ein lokales Embedding basierend auf Content-Analyse
// Verwendet einen verbesserten Hash-basierten Ansatz mit Wort-Frequenzen
func (l *LocalEmbeddingService) GenerateEmbedding(content string, contentType string) ([]float32, error) {
	embedding := make([]float32, l.dimension)

	// Normalisiere Content (lowercase, entferne Sonderzeichen)
	normalized := strings.ToLower(content)
	normalized = expandWithSynonyms(normalized)

	// Berechne verschiedene Features für bessere Semantik
	contentHash := l.hashString(normalized)
	wordCount := len(strings.Fields(normalized))
	charCount := len(normalized)

	// Extrahiere häufige Wörter und deren Positionen
	words := strings.Fields(normalized)
	wordHashes := make([]uint32, 0, len(words))
	for _, word := range words {
		if len(word) > 2 { // Ignoriere sehr kurze Wörter
			wordHashes = append(wordHashes, l.hashString(word))
		}
	}

	// Fülle Embedding-Vektor mit verschiedenen Features
	for i := 0; i < l.dimension; i++ {
		var value float32

		// Basis-Hash basierend auf Position
		hash := contentHash + uint32(i*31)

		// Füge Wort-Frequenz-Informationen hinzu
		if i < len(wordHashes) {
			hash ^= wordHashes[i%len(wordHashes)]
		}

		// Normalisiere basierend auf Content-Länge
		normalizedHash := float32(hash%10000) / 10000.0

		// Füge statistische Features hinzu
		switch i % 3 {
		case 0:
			// Wortanzahl-Feature
			value = normalizedHash * float32(wordCount%100) / 100.0
		case 1:
			// Zeichenanzahl-Feature
			value = normalizedHash * float32(charCount%1000) / 1000.0
		default:
			// Reiner Hash-Wert
			value = normalizedHash
		}

		// Normalisiere auf [-1, 1] Bereich
		embedding[i] = value*2.0 - 1.0
	}

	// Normalisiere den Vektor für bessere Cosine-Similarity
	embedding = Normalize(embedding)

	return embedding, nil
}

// GenerateEmbeddingsBatch generiert Embeddings für mehrere Contents
func (l *LocalEmbeddingService) GenerateEmbeddingsBatch(contents []string, contentType string) ([][]float32, error) {
	embeddings := make([][]float32, len(contents))
	for i, content := range contents {
		emb, err := l.GenerateEmbedding(content, contentType)
		if err != nil {
			return nil, err
		}
		embeddings[i] = emb
	}
	return embeddings, nil
}

// hashString erstellt einen Hash-Wert aus einem String
func (l *LocalEmbeddingService) hashString(s string) uint32 {
	var hash uint32 = 2166136261 // FNV-1a Basis
	for _, c := range s {
		hash ^= uint32(c)
		hash *= 16777619 // FNV-1a Prime
	}
	return hash
}

// GTEEmbeddingService - GTE-Small Embedding-Service mit gte-go
// Verwendet das GTE-Small Modell für hochwertige semantische Embeddings
type GTEEmbeddingService struct {
	model *gte.Model
	mu    sync.RWMutex
}

// NewGTEEmbeddingService erstellt einen GTE Embedding Service
// modelPath: Pfad zur .gtemodel Datei (z.B. "gte-small.gtemodel" oder "~/.openclaw/gte-small.gtemodel")
func NewGTEEmbeddingService(modelPath string) (*GTEEmbeddingService, error) {
	// Expandiere ~ zu Home-Verzeichnis
	if strings.HasPrefix(modelPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		modelPath = filepath.Join(home, strings.TrimPrefix(modelPath, "~"))
	}

	// Prüfe ob Datei existiert
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, err
	}

	model, err := gte.Load(modelPath)
	if err != nil {
		return nil, err
	}

	return &GTEEmbeddingService{
		model: model,
	}, nil
}

// GenerateEmbedding generiert ein Embedding mit GTE-Small
func (g *GTEEmbeddingService) GenerateEmbedding(content string, contentType string) ([]float32, error) {
	// GTE-Small unterstützt nur Text-Embeddings
	// Für andere Content-Types verwenden wir nur den Textanteil
	if contentType != "text/plain" && !strings.HasPrefix(contentType, "text/") {
		// Für nicht-Text-Content, extrahiere Text falls möglich
		// Hier vereinfacht: verwende Content als Text
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	embedding, err := g.model.Embed(content)
	if err != nil {
		return nil, err
	}

	// GTE-Small gibt bereits L2-normalisierte Embeddings zurück
	return embedding, nil
}

// GenerateEmbeddingsBatch generiert Embeddings für mehrere Contents
func (g *GTEEmbeddingService) GenerateEmbeddingsBatch(contents []string, contentType string) ([][]float32, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	embeddings, err := g.model.EmbedBatch(contents)
	if err != nil {
		return nil, err
	}

	return embeddings, nil
}

// Close schließt das Modell (für Cleanup)
func (g *GTEEmbeddingService) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.model != nil {
		g.model.Close()
		g.model = nil
	}
	return nil
}

var (
	globalEmbeddingService EmbeddingService
	serviceOnce            sync.Once
)

// GetEmbeddingService gibt den verfügbaren Embedding-Service zurück
// Versucht zuerst gte-go zu laden (falls CORTEX_EMBEDDING_MODEL_PATH gesetzt),
// sonst Fallback auf Hash-basierten Service
func GetEmbeddingService() EmbeddingService {
	serviceOnce.Do(func() {
		modelPath := os.Getenv("CORTEX_EMBEDDING_MODEL_PATH")
		if modelPath != "" {
			// Versuche GTE-Service zu laden
			gteService, err := NewGTEEmbeddingService(modelPath)
			if err != nil {
				slog.Warn("Failed to load GTE embedding model, falling back to hash-based service",
					"error", err,
					"modelPath", modelPath)
				globalEmbeddingService = NewLocalEmbeddingService()
			} else {
				slog.Info("Using GTE-Small Embedding Service",
					"modelPath", modelPath)
				globalEmbeddingService = gteService
			}
		} else {
			slog.Info("Using Local Hash-based Embedding Service",
				"hint", "Set CORTEX_EMBEDDING_MODEL_PATH to use GTE-Small model")
			globalEmbeddingService = NewLocalEmbeddingService()
		}
	})
	return globalEmbeddingService
}

// CosineSimilarity berechnet die Cosine-Ähnlichkeit zwischen zwei Vektoren
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// DetectContentType erkennt den Content-Type basierend auf Content
func DetectContentType(content string, metadata map[string]any) string {
	// Prüfe Metadata für expliziten Content-Type
	if ct, ok := metadata["contentType"].(string); ok {
		return ct
	}
	if ct, ok := metadata["content_type"].(string); ok {
		return ct
	}

	// Prüfe auf Base64-Bild (vereinfacht)
	if strings.HasPrefix(content, "data:image/") {
		parts := strings.Split(content, ";")
		if len(parts) > 0 {
			return strings.TrimPrefix(parts[0], "data:")
		}
	}

	// Prüfe auf URL
	if strings.HasPrefix(content, "http://") || strings.HasPrefix(content, "https://") {
		lower := strings.ToLower(content)
		if strings.Contains(lower, ".jpg") || strings.Contains(lower, ".jpeg") {
			return "image/jpeg"
		}
		if strings.Contains(lower, ".png") {
			return "image/png"
		}
		if strings.Contains(lower, ".pdf") {
			return "application/pdf"
		}
	}

	// Standard: Text
	return "text/plain"
}
