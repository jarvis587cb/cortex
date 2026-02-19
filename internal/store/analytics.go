package store

import (
	"time"

	"cortex/internal/models"
)

// AnalyticsData represents analytics data for a tenant
type AnalyticsData struct {
	TenantID          string                 `json:"tenant_id"`
	AppID             string                 `json:"app_id"`
	ExternalUserID    string                 `json:"external_user_id"`
	TotalMemories     int64                  `json:"total_memories"`
	TotalBundles      int64                  `json:"total_bundles"`
	MemoriesWithEmbeddings int64             `json:"memories_with_embeddings"`
	MemoriesByType    map[string]int64        `json:"memories_by_type"`
	MemoriesByBundle  map[int64]int64        `json:"memories_by_bundle"`
	RecentActivity    []ActivityEntry         `json:"recent_activity"`
	StorageStats      StorageStats            `json:"storage_stats"`
	TimeRange         TimeRange               `json:"time_range"`
}

// ActivityEntry represents a recent activity entry
type ActivityEntry struct {
	Type      string    `json:"type"`      // "memory.created", "bundle.created", etc.
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
}

// StorageStats represents storage statistics
type StorageStats struct {
	TotalSize      int64 `json:"total_size"`       // Approximate database size
	MemoriesCount  int64 `json:"memories_count"`
	BundlesCount   int64 `json:"bundles_count"`
	WebhooksCount  int64 `json:"webhooks_count"`
}

// TimeRange represents a time range for analytics
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// GetAnalytics retrieves analytics data for a tenant
func (s *CortexStore) GetAnalytics(appID, externalUserID string, days int) (*AnalyticsData, error) {
	if days <= 0 {
		days = 30 // Default: last 30 days
	}

	startTime := time.Now().AddDate(0, 0, -days)
	endTime := time.Now()

	// Total memories
	var totalMemories int64
	s.db.Model(&models.Memory{}).
		Where("app_id = ? AND external_user_id = ?", appID, externalUserID).
		Count(&totalMemories)

	// Total bundles
	var totalBundles int64
	s.db.Model(&models.Bundle{}).
		Where("app_id = ? AND external_user_id = ?", appID, externalUserID).
		Count(&totalBundles)

	// Memories with embeddings
	var memoriesWithEmbeddings int64
	s.db.Model(&models.Memory{}).
		Where("app_id = ? AND external_user_id = ? AND embedding != ''", appID, externalUserID).
		Count(&memoriesWithEmbeddings)

	// Memories by type
	typeCounts := make(map[string]int64)
	var typeResults []struct {
		Type  string `gorm:"column:type"`
		Count int64  `gorm:"column:count"`
	}
	s.db.Model(&models.Memory{}).
		Select("type, COUNT(*) as count").
		Where("app_id = ? AND external_user_id = ?", appID, externalUserID).
		Group("type").
		Scan(&typeResults)
	for _, r := range typeResults {
		typeCounts[r.Type] = r.Count
	}

	// Memories by bundle
	bundleCounts := make(map[int64]int64)
	var bundleResults []struct {
		BundleID *int64 `gorm:"column:bundle_id"`
		Count    int64  `gorm:"column:count"`
	}
	s.db.Model(&models.Memory{}).
		Select("bundle_id, COUNT(*) as count").
		Where("app_id = ? AND external_user_id = ? AND bundle_id IS NOT NULL", appID, externalUserID).
		Group("bundle_id").
		Scan(&bundleResults)
	for _, r := range bundleResults {
		if r.BundleID != nil {
			bundleCounts[*r.BundleID] = r.Count
		}
	}

	// Recent activity (last 50 entries)
	var recentMemories []models.Memory
	s.db.Model(&models.Memory{}).
		Where("app_id = ? AND external_user_id = ? AND created_at >= ?", appID, externalUserID, startTime).
		Order("created_at DESC").
		Limit(25).
		Find(&recentMemories)

	var recentBundles []models.Bundle
	s.db.Model(&models.Bundle{}).
		Where("app_id = ? AND external_user_id = ? AND created_at >= ?", appID, externalUserID, startTime).
		Order("created_at DESC").
		Limit(25).
		Find(&recentBundles)

	// Combine activities
	activities := make([]ActivityEntry, 0, len(recentMemories)+len(recentBundles))
	for _, m := range recentMemories {
		activities = append(activities, ActivityEntry{
			Type:      "memory.created",
			ID:        m.ID,
			Timestamp: m.CreatedAt,
		})
	}
	for _, b := range recentBundles {
		activities = append(activities, ActivityEntry{
			Type:      "bundle.created",
			ID:        b.ID,
			Timestamp: b.CreatedAt,
		})
	}

	// Sort by timestamp (most recent first)
	// Simple bubble sort for small arrays
	for i := 0; i < len(activities)-1; i++ {
		for j := i + 1; j < len(activities); j++ {
			if activities[i].Timestamp.Before(activities[j].Timestamp) {
				activities[i], activities[j] = activities[j], activities[i]
			}
		}
	}

	// Limit to 50 most recent
	if len(activities) > 50 {
		activities = activities[:50]
	}

	// Storage stats
	var webhooksCount int64
	s.db.Model(&models.Webhook{}).
		Where("app_id = ? OR app_id = ?", appID, "").
		Count(&webhooksCount)

	storageStats := StorageStats{
		MemoriesCount: totalMemories,
		BundlesCount:  totalBundles,
		WebhooksCount: webhooksCount,
	}

	return &AnalyticsData{
		TenantID:             appID + ":" + externalUserID,
		AppID:                appID,
		ExternalUserID:       externalUserID,
		TotalMemories:        totalMemories,
		TotalBundles:         totalBundles,
		MemoriesWithEmbeddings: memoriesWithEmbeddings,
		MemoriesByType:       typeCounts,
		MemoriesByBundle:     bundleCounts,
		RecentActivity:       activities,
		StorageStats:         storageStats,
		TimeRange: TimeRange{
			Start: startTime,
			End:   endTime,
		},
	}, nil
}

// GetGlobalAnalytics retrieves global analytics (all tenants)
func (s *CortexStore) GetGlobalAnalytics(days int) (*AnalyticsData, error) {
	if days <= 0 {
		days = 30
	}

	startTime := time.Now().AddDate(0, 0, -days)
	endTime := time.Now()

	var totalMemories int64
	s.db.Model(&models.Memory{}).Count(&totalMemories)

	var totalBundles int64
	s.db.Model(&models.Bundle{}).Count(&totalBundles)

	var memoriesWithEmbeddings int64
	s.db.Model(&models.Memory{}).
		Where("embedding != ''").
		Count(&memoriesWithEmbeddings)

	var webhooksCount int64
	s.db.Model(&models.Webhook{}).Count(&webhooksCount)

	storageStats := StorageStats{
		MemoriesCount: totalMemories,
		BundlesCount:  totalBundles,
		WebhooksCount: webhooksCount,
	}

	return &AnalyticsData{
		TenantID:             "global",
		AppID:                "",
		ExternalUserID:       "",
		TotalMemories:        totalMemories,
		TotalBundles:         totalBundles,
		MemoriesWithEmbeddings: memoriesWithEmbeddings,
		MemoriesByType:       make(map[string]int64),
		MemoriesByBundle:     make(map[int64]int64),
		RecentActivity:       []ActivityEntry{},
		StorageStats:         storageStats,
		TimeRange: TimeRange{
			Start: startTime,
			End:   endTime,
		},
	}, nil
}
