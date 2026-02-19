package models

import "time"

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

// Neutron-compatible Seeds API Types

type StoreSeedRequest struct {
	AppID          string         `json:"appId"`
	ExternalUserID string         `json:"externalUserId"`
	Content        string         `json:"content"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	BundleID       *int64         `json:"bundleId,omitempty"`
}

func (r *StoreSeedRequest) GetAppID() string        { return r.AppID }
func (r *StoreSeedRequest) GetExternalUserID() string { return r.ExternalUserID }

type StoreSeedResponse struct {
	ID      int64  `json:"id"`
	Message string `json:"message"`
}

type QuerySeedRequest struct {
	AppID          string `json:"appId"`
	ExternalUserID string `json:"externalUserId"`
	Query          string `json:"query"`
	Limit          int    `json:"limit,omitempty"`
	BundleID       *int64 `json:"bundleId,omitempty"`
}

func (r *QuerySeedRequest) GetAppID() string        { return r.AppID }
func (r *QuerySeedRequest) GetExternalUserID() string { return r.ExternalUserID }

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
	AppID          string `json:"appId"`
	ExternalUserID string `json:"externalUserId"`
	Name           string `json:"name"`
}

func (r *CreateBundleRequest) GetAppID() string        { return r.AppID }
func (r *CreateBundleRequest) GetExternalUserID() string { return r.ExternalUserID }

type BundleResponse struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	AppID          string    `json:"app_id"`
	ExternalUserID string    `json:"external_user_id"`
	CreatedAt      time.Time `json:"created_at"`
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
