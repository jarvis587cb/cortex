package models

import (
	"cortex/internal/helpers"
	"strings"
	"time"
)

// Database Models

type Memory struct {
	ID             int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Type           string         `gorm:"not null;default:'semantic'" json:"type"`
	Content        string         `gorm:"not null" json:"content"`
	Entity         string         `json:"entity,omitempty"`
	Tags           string         `json:"tags,omitempty"`
	Importance     int            `gorm:"not null;default:5" json:"importance"`
	AppID          string         `gorm:"column:app_id;not null;default:'openclaw';index" json:"app_id,omitempty"`
	ExternalUserID string         `gorm:"column:external_user_id;not null;default:'default';index" json:"external_user_id,omitempty"`
	BundleID       *int64         `gorm:"column:bundle_id;index" json:"bundle_id,omitempty"`
	Metadata       string         `gorm:"type:text" json:"-"`
	MetadataMap    map[string]any `gorm:"-" json:"metadata,omitempty"`
	Embedding      string         `gorm:"type:text" json:"-"` // JSON-encoded []float32
	ContentType    string         `gorm:"column:content_type;default:'text/plain'" json:"content_type,omitempty"`
	CreatedAt      time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// NewMemoryFromRememberRequest creates a Memory from RememberRequest
func NewMemoryFromRememberRequest(req *RememberRequest) *Memory {
	mem := &Memory{
		Type:       req.Type,
		Content:    req.Content,
		Entity:     req.Entity,
		Tags:       req.Tags,
		Importance: req.Importance,
	}
	if mem.Type == "" {
		mem.Type = helpers.DefaultMemType
	}
	if mem.Importance == 0 {
		mem.Importance = helpers.DefaultImportance
	}
	return mem
}

// NewMemoryFromStoreSeedRequest creates a Memory from StoreSeedRequest
func NewMemoryFromStoreSeedRequest(req *StoreSeedRequest, appID, externalUserID string) *Memory {
	return &Memory{
		Type:           "semantic",
		Content:        req.Content,
		AppID:          appID,
		ExternalUserID: externalUserID,
		BundleID:       req.BundleID,
		Metadata:       helpers.MarshalMetadata(req.Metadata),
		Importance:     5,
	}
}

type Entity struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string         `gorm:"uniqueIndex;not null" json:"name"`
	Data      string         `gorm:"type:text" json:"-"`
	DataMap   map[string]any `gorm:"-" json:"data"`
	CreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

type Relation struct {
	ID        int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	From      string     `gorm:"column:from_entity;not null" json:"from"`
	To        string     `gorm:"column:to_entity;not null" json:"to"`
	Type      string     `gorm:"column:type;not null" json:"type"`
	ValidFrom *time.Time `gorm:"column:valid_from" json:"valid_from,omitempty"`
	ValidTo   *time.Time `gorm:"column:valid_to" json:"valid_to,omitempty"`
	CreatedAt time.Time  `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

type Bundle struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name           string    `gorm:"not null" json:"name"`
	AppID          string    `gorm:"column:app_id;not null;index" json:"app_id"`
	ExternalUserID string    `gorm:"column:external_user_id;not null;index" json:"external_user_id"`
	CreatedAt      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

type Webhook struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	URL       string    `gorm:"not null" json:"url"`
	Events    string    `gorm:"not null" json:"events"` // Comma-separated event types
	Secret    string    `gorm:"not null" json:"-"`      // Webhook secret for signing
	AppID     string    `gorm:"column:app_id;index" json:"app_id,omitempty"`
	Active    bool      `gorm:"default:true" json:"active"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// AgentContext: session state / conversation history (Neutron-compatible)
// MemoryType: episodic, semantic, procedural, working
type AgentContext struct {
	ID             int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	AppID          string         `gorm:"column:app_id;not null;index" json:"app_id"`
	ExternalUserID string         `gorm:"column:external_user_id;not null;index" json:"external_user_id"`
	AgentID        string         `gorm:"column:agent_id;not null;index" json:"agent_id"`
	MemoryType     string         `gorm:"column:memory_type;not null;index" json:"memory_type"` // episodic, semantic, procedural, working
	Payload        string         `gorm:"type:text;not null" json:"-"`                          // JSON
	PayloadMap     map[string]any `gorm:"-" json:"payload,omitempty"`
	Tags           string         `gorm:"type:text" json:"-"` // optional comma-separated or JSON array
	CreatedAt      time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

type Stats struct {
	Memories  int64 `json:"memories"`
	Entities  int64 `json:"entities"`
	Relations int64 `json:"relations"`
}

// Request/Response Types

type RememberRequest struct {
	Content    string `json:"content"`
	Type       string `json:"type,omitempty"`
	Entity     string `json:"entity,omitempty"`
	Tags       string `json:"tags,omitempty"`
	Importance int    `json:"importance,omitempty"`
}

type RememberResponse struct {
	ID int64 `json:"id"`
}

type FactRequest struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type RelationRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

// TenantRequest provides common tenant parameter fields and getters
type TenantRequest struct {
	AppID          string `json:"appId"`
	ExternalUserID string `json:"externalUserId"`
}

func (r *TenantRequest) GetAppID() string          { return r.AppID }
func (r *TenantRequest) GetExternalUserID() string { return r.ExternalUserID }

// Neutron-compatible Seeds API Types

type StoreSeedRequest struct {
	TenantRequest
	Content  string         `json:"content"`
	Metadata map[string]any `json:"metadata,omitempty"`
	BundleID *int64         `json:"bundleId,omitempty"`
}

type StoreSeedResponse struct {
	ID      int64  `json:"id"`
	Message string `json:"message"`
}

type QuerySeedRequest struct {
	TenantRequest
	Query     string  `json:"query"`
	Limit     int     `json:"limit,omitempty"`
	BundleID  *int64  `json:"bundleId,omitempty"`
	Threshold float64 `json:"threshold,omitempty"` // 0-1, default 0; only return results with similarity >= threshold
	SeedIDs   []int64 `json:"seedIds,omitempty"`   // optional: limit search to these memory IDs
}

type QuerySeedResult struct {
	ID         int64          `json:"id"`
	Content    string         `json:"content"`
	Metadata   map[string]any `json:"metadata"`
	CreatedAt  time.Time      `json:"created_at"`
	Similarity float64        `json:"similarity"`
}

type DeleteSeedResponse struct {
	Message string `json:"message"`
	ID      int64  `json:"id"`
}

// Bundle API Types

type CreateBundleRequest struct {
	TenantRequest
	Name string `json:"name"`
}

type BundleResponse struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	AppID          string    `json:"app_id"`
	ExternalUserID string    `json:"external_user_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// ToBundleResponse converts a Bundle model to BundleResponse
func (b *Bundle) ToBundleResponse() BundleResponse {
	return BundleResponse{
		ID:             b.ID,
		Name:           b.Name,
		AppID:          b.AppID,
		ExternalUserID: b.ExternalUserID,
		CreatedAt:      b.CreatedAt,
	}
}

// Webhook API Types

type CreateWebhookRequest struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret,omitempty"`
	AppID  string   `json:"appId,omitempty"`
}

type WebhookResponse struct {
	ID        int64     `json:"id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	AppID     string    `json:"app_id,omitempty"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Agent Context API Types (Neutron-compatible)

type CreateAgentContextRequest struct {
	TenantRequest
	AgentID    string         `json:"agentId"`
	MemoryType string         `json:"memoryType"` // episodic, semantic, procedural, working
	Payload    map[string]any `json:"payload"`
	Tags       []string       `json:"tags,omitempty"`
}

type AgentContextResponse struct {
	ID             int64          `json:"id"`
	AppID          string         `json:"app_id"`
	ExternalUserID string         `json:"external_user_id"`
	AgentID        string         `json:"agent_id"`
	MemoryType     string         `json:"memory_type"`
	Payload        map[string]any `json:"payload"`
	Tags           []string       `json:"tags,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// ToWebhookResponse converts a Webhook model to WebhookResponse
// eventsStr should be the comma-separated events string from the model
func (w *Webhook) ToWebhookResponse(eventsStr string) WebhookResponse {
	events := make([]string, 0)
	if eventsStr != "" {
		for _, e := range strings.Split(eventsStr, ",") {
			events = append(events, strings.TrimSpace(e))
		}
	}
	return WebhookResponse{
		ID:        w.ID,
		URL:       w.URL,
		Events:    events,
		AppID:     w.AppID,
		Active:    w.Active,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}
