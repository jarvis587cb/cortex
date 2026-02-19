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
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Log error but don't fail - response already started
		// In production, consider logging this error
	}
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

// TenantParamExtractor interface für Request-Types mit Tenant-Parametern
type TenantParamExtractor interface {
	GetAppID() string
	GetExternalUserID() string
}

// ExtractTenantParams extrahiert appID und externalUserID aus Query-Parametern oder Request-Body
// Query-Parameter haben Priorität (Neutron-kompatibel), Fallback zu Request-Body
func ExtractTenantParams(r *http.Request, req TenantParamExtractor) (appID, externalUserID string) {
	appID = GetQueryParam(r, "appId")
	externalUserID = GetQueryParam(r, "externalUserId")

	// Fallback zu Request-Body wenn Query-Parameter leer
	if appID == "" && req != nil {
		appID = req.GetAppID()
	}
	if externalUserID == "" && req != nil {
		externalUserID = req.GetExternalUserID()
	}

	return appID, externalUserID
}

// WriteError writes an error response as JSON
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{
		"error": message,
	})
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
	payload, err := json.Marshal(metadata)
	if err != nil {
		return ""
	}
	return string(payload)
}

func UnmarshalMetadata(metadataJSON string) map[string]any {
	if metadataJSON == "" {
		return map[string]any{}
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return map[string]any{}
	}
	return metadata
}

func MarshalEntityData(data map[string]any) string {
	payload, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(payload)
}

func UnmarshalEntityData(dataJSON string) map[string]any {
	if dataJSON == "" {
		return map[string]any{}
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return map[string]any{}
	}
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
