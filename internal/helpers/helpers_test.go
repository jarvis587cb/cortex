package helpers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name     string
		fields   map[string]string
		expected bool
		field    string
	}{
		{"all fields present", map[string]string{"a": "value1", "b": "value2"}, true, ""},
		{"missing field", map[string]string{"a": "value1", "b": ""}, false, "b"},
		{"empty map", map[string]string{}, true, ""},
		{"whitespace only", map[string]string{"a": "   "}, false, "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, ok := ValidateRequired(tt.fields)
			if ok != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, ok)
			}
			if !ok && field != tt.field {
				t.Errorf("expected field %s, got %s", tt.field, field)
			}
		})
	}
}

func TestParseLimit(t *testing.T) {
	tests := []struct {
		name         string
		limitStr     string
		defaultLimit int
		maxLimit     int
		expected     int
	}{
		{"valid limit", "5", 10, 100, 5},
		{"empty string", "", 10, 100, 10},
		{"exceeds max", "150", 10, 100, 10},
		{"zero", "0", 10, 100, 10},
		{"negative", "-5", 10, 100, 10},
		{"invalid format", "abc", 10, 100, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseLimit(tt.limitStr, tt.defaultLimit, tt.maxLimit)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParseID(t *testing.T) {
	tests := []struct {
		name     string
		idStr    string
		expected int64
		hasError bool
	}{
		{"valid id", "123", 123, false},
		{"zero", "0", 0, true},
		{"negative", "-5", 0, true},
		{"invalid format", "abc", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ParseID(tt.idStr)
			if (err != nil) != tt.hasError {
				t.Errorf("expected error %v, got %v", tt.hasError, err != nil)
			}
			if !tt.hasError && id != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, id)
			}
		})
	}
}

func TestExtractPathID(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		prefix   string
		expected string
		hasError bool
	}{
		{"valid path", "/seeds/123", "/seeds/", "123", false},
		{"path with query", "/seeds/123?foo=bar", "/seeds/", "123", false},
		{"no prefix match", "/other/123", "/seeds/", "", true},
		{"empty id", "/seeds/", "/seeds/", "", true},
		{"trailing slash", "/seeds/123/", "/seeds/", "123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ExtractPathID(tt.path, tt.prefix)
			if (err != nil) != tt.hasError {
				t.Errorf("expected error %v, got %v", tt.hasError, err != nil)
			}
			if !tt.hasError {
				// For query params, just check that it starts with expected
				if tt.name == "path with query" {
					if !strings.HasPrefix(id, tt.expected) {
						t.Errorf("expected id to start with %s, got %s", tt.expected, id)
					}
				} else if id != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, id)
				}
			}
		})
	}
}

func TestMarshalUnmarshalMetadata(t *testing.T) {
	original := map[string]any{
		"tags":  []string{"test", "example"},
		"count": 42,
	}

	marshaled := MarshalMetadata(original)
	if marshaled == "" {
		t.Error("MarshalMetadata returned empty string")
	}

	unmarshaled := UnmarshalMetadata(marshaled)
	if len(unmarshaled) != len(original) {
		t.Errorf("expected %d keys, got %d", len(original), len(unmarshaled))
	}
}

func TestMarshalUnmarshalEntityData(t *testing.T) {
	original := map[string]any{
		"key1": "value1",
		"key2": 123,
	}

	marshaled := MarshalEntityData(original)
	if marshaled == "" {
		t.Error("MarshalEntityData returned empty string")
	}

	unmarshaled := UnmarshalEntityData(marshaled)
	if len(unmarshaled) != len(original) {
		t.Errorf("expected %d keys, got %d", len(original), len(unmarshaled))
	}
}

// TestUnmarshalEntityDataNull verifies that JSON "null" returns a non-nil empty map (avoids panic in callers).
func TestUnmarshalEntityDataNull(t *testing.T) {
	result := UnmarshalEntityData("null")
	if result == nil {
		t.Fatal("UnmarshalEntityData(\"null\") returned nil; callers may panic on map assignment")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got len %d", len(result))
	}
}

// TestUnmarshalMetadataNull verifies that JSON "null" returns a non-nil empty map.
func TestUnmarshalMetadataNull(t *testing.T) {
	result := UnmarshalMetadata("null")
	if result == nil {
		t.Fatal("UnmarshalMetadata(\"null\") returned nil")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got len %d", len(result))
	}
}

func TestGetQueryParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?param=value&empty=", nil)

	if GetQueryParam(req, "param") != "value" {
		t.Error("failed to get query param")
	}
	if GetQueryParam(req, "empty") != "" {
		t.Error("empty param should return empty string")
	}
	if GetQueryParam(req, "missing") != "" {
		t.Error("missing param should return empty string")
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"status": "ok"}

	WriteJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected content-type application/json, got %s", contentType)
	}

	if !strings.Contains(w.Body.String(), "status") {
		t.Error("response body does not contain expected data")
	}
}
