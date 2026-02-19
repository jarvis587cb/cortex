package embeddings

import (
	"encoding/json"
	"fmt"
)

// Vector represents an embedding vector
type Vector []float32

// MarshalJSON serializes vector to JSON
func (v Vector) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float32(v))
}

// UnmarshalJSON deserializes vector from JSON
func (v *Vector) UnmarshalJSON(data []byte) error {
	var floats []float32
	if err := json.Unmarshal(data, &floats); err != nil {
		return err
	}
	*v = Vector(floats)
	return nil
}

// EncodeVector encodes a vector to JSON string for storage
func EncodeVector(v []float32) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to encode vector: %w", err)
	}
	return string(data), nil
}

// DecodeVector decodes a vector from JSON string
func DecodeVector(s string) ([]float32, error) {
	if s == "" {
		return nil, nil
	}
	var v []float32
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, fmt.Errorf("failed to decode vector: %w", err)
	}
	return v, nil
}

// Normalize normalizes a vector to unit length
func Normalize(v []float32) []float32 {
	var sum float64
	for _, val := range v {
		sum += float64(val * val)
	}
	if sum == 0 {
		return v
	}
	norm := sqrt(sum)
	result := make([]float32, len(v))
	for i, val := range v {
		result[i] = float32(float64(val) / norm)
	}
	return result
}
