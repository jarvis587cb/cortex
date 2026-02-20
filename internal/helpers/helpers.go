package helpers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// Constants
const (
	DefaultPort          = "9123"
	DefaultDBName        = "cortex.db"
	DefaultMemType       = "semantic"
	DefaultImportance    = 5
	DefaultLimit         = 10
	MaxLimit             = 100
	DefaultQueryLimit    = 5   // Default limit for query operations
	DefaultAnalyticsDays = 30  // Default days for analytics queries
	DefaultSimilarity    = 0.5 // Default similarity score
	TextMatchSimilarity  = 0.8 // Similarity score for text matches
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

// IsNotFoundError checks if an error is a "not found" error (gorm.ErrRecordNotFound)
func IsNotFoundError(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// HandleInternalError logs an error and writes an internal server error response
// This is a convenience function for the common pattern of logging and returning 500
func HandleInternalError(w http.ResponseWriter, logger func(msg string, args ...any), msg string, args ...any) {
	logger(msg, args...)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

// HandleInternalErrorSlog logs an error using slog and writes an internal server error response
func HandleInternalErrorSlog(w http.ResponseWriter, msg string, args ...any) {
	slog.Error(msg, args...)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

// NewSuccessResponse creates a standardized success response with ID and message
func NewSuccessResponse(id int64, message string) map[string]interface{} {
	return map[string]interface{}{
		"id":      id,
		"message": message,
	}
}

// ExtractAndParseID extracts ID from path and parses it to int64
// Returns the parsed ID or writes error response and returns false
func ExtractAndParseID(w http.ResponseWriter, path, prefix string) (int64, bool) {
	idStr, err := ExtractPathID(path, prefix)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return 0, false
	}

	id, err := ParseID(idStr)
	if err != nil {
		http.Error(w, "invalid id format", http.StatusBadRequest)
		return 0, false
	}

	return id, true
}

// HandleNotFoundError handles not found errors consistently
// Returns true if error was handled (not found), false otherwise
func HandleNotFoundError(w http.ResponseWriter, err error, resourceName string) bool {
	if IsNotFoundError(err) {
		http.Error(w, resourceName+" not found", http.StatusNotFound)
		return true
	}
	return false
}

// ParseJSONBodyOrError parses JSON body and writes error response if parsing fails
// Returns true if parsing succeeded, false otherwise (error already written)
func ParseJSONBodyOrError(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := ParseJSONBody(r, v); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return false
	}
	return true
}

// ValidateNotEmpty validates that a string is not empty after trimming
// Returns true if valid, false otherwise
func ValidateNotEmpty(value, fieldName string) bool {
	return strings.TrimSpace(value) != ""
}

// ValidateWebhookURL returns an error if the URL is empty, unparseable, or not http/https.
func ValidateWebhookURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return errors.New("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("url must use http or https scheme")
	}
	if u.Host == "" {
		return errors.New("url must have a host")
	}
	return nil
}

// ValidateTenantParams validates tenant parameters and writes error response if invalid
// Returns true if valid, false otherwise (error already written)
func ValidateTenantParams(w http.ResponseWriter, r *http.Request, req TenantParamExtractor, isQueryParam bool) (appID, externalUserID string, ok bool) {
	appID, externalUserID = ExtractTenantParams(r, req)

	fields := map[string]string{
		"appId":          appID,
		"externalUserId": externalUserID,
	}

	if field, valid := ValidateRequired(fields); !valid {
		msg := "missing required field: " + field
		if isQueryParam {
			msg = "missing required query parameter: " + field
		}
		http.Error(w, msg, http.StatusBadRequest)
		return "", "", false
	}

	return appID, externalUserID, true
}

// ValidateTenantParamsWithFields validates tenant parameters plus additional fields
// Returns true if valid, false otherwise (error already written)
func ValidateTenantParamsWithFields(w http.ResponseWriter, r *http.Request, req TenantParamExtractor, additionalFields map[string]string, isQueryParam bool) (appID, externalUserID string, ok bool) {
	appID, externalUserID = ExtractTenantParams(r, req)

	fields := map[string]string{
		"appId":          appID,
		"externalUserId": externalUserID,
	}
	for k, v := range additionalFields {
		fields[k] = v
	}

	if field, valid := ValidateRequired(fields); !valid {
		msg := "missing required field: " + field
		if isQueryParam {
			msg = "missing required query parameter: " + field
		}
		http.Error(w, msg, http.StatusBadRequest)
		return "", "", false
	}

	return appID, externalUserID, true
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
	if metadata == nil {
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
	if data == nil {
		return map[string]any{}
	}
	return data
}

// SanitizeFilenameForHeader removes characters that could break HTTP header values (e.g. Content-Disposition filename).
// Removes double-quote, backslash, newline, carriage return.
func SanitizeFilenameForHeader(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '"', '\\', '\n', '\r':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SafeJSONPathKey reports whether key is safe to use in a simple SQLite json_extract path ("$."+key).
// Allows only letters, digits, underscore, hyphen, and dot to avoid path injection or broken paths.
func SafeJSONPathKey(key string) bool {
	if key == "" {
		return false
	}
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '_', r == '-', r == '.':
		default:
			return false
		}
	}
	return true
}

// ValidateBackupPath rejects paths that could cause path traversal (e.g. ".." or absolute paths).
// Use for backup/restore path parameters.
func ValidateBackupPath(path string) error {
	if path == "" {
		return errors.New("path is required")
	}
	cleaned := filepath.Clean(path)
	if filepath.IsAbs(cleaned) {
		return errors.New("absolute paths are not allowed")
	}
	if strings.Contains(cleaned, "..") {
		return errors.New("path must not contain '..'")
	}
	return nil
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
