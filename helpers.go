package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// Constants
const (
	DefaultPort     = "9123"
	DefaultDBName   = "cortex.db"
	DefaultMemType  = "semantic"
	DefaultImportance = 5
	DefaultLimit    = 10
	MaxLimit        = 100
)

// JSON Helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func parseJSONBody(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// Validation Helpers

func validateRequired(fields map[string]string) (string, bool) {
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return field, false
		}
	}
	return "", true
}

func parseLimit(limitStr string, defaultLimit, maxLimit int) int {
	if limitStr == "" {
		return defaultLimit
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > maxLimit {
		return defaultLimit
	}
	return limit
}

func parseID(idStr string) (int64, error) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return 0, err
	}
	return id, nil
}

// Metadata Mapping Helpers

func mapMemoryMetadata(memories []Memory) {
	for i := range memories {
		if memories[i].Metadata != "" {
			_ = json.Unmarshal([]byte(memories[i].Metadata), &memories[i].MetadataMap)
		}
	}
}

func mapEntityData(entities []Entity) {
	for i := range entities {
		data := map[string]any{}
		if entities[i].Data != "" {
			_ = json.Unmarshal([]byte(entities[i].Data), &data)
		}
		entities[i].DataMap = data
	}
}

func mapEntityDataSingle(entity *Entity) {
	data := map[string]any{}
	if entity.Data != "" {
		_ = json.Unmarshal([]byte(entity.Data), &data)
	}
	entity.DataMap = data
}

func marshalMetadata(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}
	payload, _ := json.Marshal(metadata)
	return string(payload)
}

func unmarshalMetadata(metadataJSON string) map[string]any {
	if metadataJSON == "" {
		return map[string]any{}
	}
	var metadata map[string]any
	_ = json.Unmarshal([]byte(metadataJSON), &metadata)
	return metadata
}

func marshalEntityData(data map[string]any) string {
	payload, _ := json.Marshal(data)
	return string(payload)
}

func unmarshalEntityData(dataJSON string) map[string]any {
	if dataJSON == "" {
		return map[string]any{}
	}
	var data map[string]any
	_ = json.Unmarshal([]byte(dataJSON), &data)
	return data
}

// Query Parameter Helpers

func getQueryParam(r *http.Request, key string) string {
	return strings.TrimSpace(r.URL.Query().Get(key))
}

func extractPathID(path, prefix string) (string, error) {
	idStr := strings.TrimPrefix(path, prefix)
	if idStr == "" || idStr == path {
		return "", errMissingID
	}
	idStr = strings.Split(idStr, "/")[0]
	if idStr == "" {
		return "", errMissingID
	}
	return idStr, nil
}

// Errors
var (
	errMissingID = &validationError{field: "id", message: "missing id in path"}
)

type validationError struct {
	field   string
	message string
}

func (e *validationError) Error() string {
	if e.field != "" {
		return e.message + ": " + e.field
	}
	return e.message
}
