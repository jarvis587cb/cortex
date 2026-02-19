package helpers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// Constants
const (
	DefaultPort       = "9123"
	DefaultDBName     = "cortex.db"
	DefaultMemType    = "semantic"
	DefaultImportance = 5
	DefaultLimit      = 10
	MaxLimit          = 100
)

// JSON Helpers

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func ParseJSONBody(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// Validation Helpers

func ValidateRequired(fields map[string]string) (string, bool) {
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return field, false
		}
	}
	return "", true
}

func ParseLimit(limitStr string, defaultLimit, maxLimit int) int {
	if limitStr == "" {
		return defaultLimit
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > maxLimit {
		return defaultLimit
	}
	return limit
}

func ParseID(idStr string) (int64, error) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, err
	}
	if id <= 0 {
		return 0, &ValidationError{Field: "id", Message: "id must be positive"}
	}
	return id, nil
}

// Metadata Mapping Helpers

func MarshalMetadata(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}
	payload, _ := json.Marshal(metadata)
	return string(payload)
}

func UnmarshalMetadata(metadataJSON string) map[string]any {
	if metadataJSON == "" {
		return map[string]any{}
	}
	var metadata map[string]any
	_ = json.Unmarshal([]byte(metadataJSON), &metadata)
	return metadata
}

func MarshalEntityData(data map[string]any) string {
	payload, _ := json.Marshal(data)
	return string(payload)
}

func UnmarshalEntityData(dataJSON string) map[string]any {
	if dataJSON == "" {
		return map[string]any{}
	}
	var data map[string]any
	_ = json.Unmarshal([]byte(dataJSON), &data)
	return data
}

// Query Parameter Helpers

func GetQueryParam(r *http.Request, key string) string {
	return strings.TrimSpace(r.URL.Query().Get(key))
}

func ExtractPathID(path, prefix string) (string, error) {
	idStr := strings.TrimPrefix(path, prefix)
	if idStr == "" || idStr == path {
		return "", ErrMissingID
	}
	// Split on "/" and "?" to handle query parameters
	idStr = strings.Split(idStr, "/")[0]
	idStr = strings.Split(idStr, "?")[0]
	if idStr == "" {
		return "", ErrMissingID
	}
	return idStr, nil
}

// Errors
var (
	ErrMissingID = &ValidationError{Field: "id", Message: "missing id in path"}
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return e.Message + ": " + e.Field
	}
	return e.Message
}
