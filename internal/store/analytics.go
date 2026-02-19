package store

import (
	"sort"
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
		days = 30 // Default: last 30 days (keep as literal since this is in store package)
	}

	startTime := time.Now().AddDate(0, 0, -days)
	endTime := time.Now()

	// Total memories, bundles, and memories with embeddings (combined query)
	var counts struct {
		TotalMemories         int64 `gorm:"column:total_memories"`
		TotalBundles          int64 `gorm:"column:total_bundles"`
		MemoriesWithEmbeddings int64 `gorm:"column:memories_with_embeddings"`
	}

	err := s.db.Raw(`
		SELECT 
			(SELECT COUNT(*) FROM memories WHERE app_id = ? AND external_user_id = ?) as total_memories,
			(SELECT COUNT(*) FROM bundles WHERE app_id = ? AND external_user_id = ?) as total_bundles,
			(SELECT COUNT(*) FROM memories WHERE app_id = ? AND external_user_id = ? AND embedding != '' AND embedding IS NOT NULL) as memories_with_embeddings
	`, appID, externalUserID, appID, externalUserID, appID, externalUserID).Scan(&counts).Error

	if err != nil {
		return nil, err
	}

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
	sort.Slice(activities, func(i, j int) bool {
		return activities[i].Timestamp.After(activities[j].Timestamp)
	})

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
		MemoriesCount: counts.TotalMemories,
		BundlesCount:  counts.TotalBundles,
		WebhooksCount: webhooksCount,
	}

	return &AnalyticsData{
		TenantID:             appID + ":" + externalUserID,
		AppID:                appID,
		ExternalUserID:       externalUserID,
		TotalMemories:        counts.TotalMemories,
		TotalBundles:         counts.TotalBundles,
		MemoriesWithEmbeddings: counts.MemoriesWithEmbeddings,
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
		days = 30 // Default: last 30 days
	}

	startTime := time.Now().AddDate(0, 0, -days)
	endTime := time.Now()

	// Combined COUNT queries for global analytics
	var counts struct {
		TotalMemories         int64 `gorm:"column:total_memories"`
		TotalBundles          int64 `gorm:"column:total_bundles"`
		MemoriesWithEmbeddings int64 `gorm:"column:memories_with_embeddings"`
		WebhooksCount         int64 `gorm:"column:webhooks_count"`
	}

	err := s.db.Raw(`
		SELECT 
			(SELECT COUNT(*) FROM memories) as total_memories,
			(SELECT COUNT(*) FROM bundles) as total_bundles,
			(SELECT COUNT(*) FROM memories WHERE embedding != '' AND embedding IS NOT NULL) as memories_with_embeddings,
			(SELECT COUNT(*) FROM webhooks) as webhooks_count
	`).Scan(&counts).Error

	if err != nil {
		return nil, err
	}

	storageStats := StorageStats{
		MemoriesCount: counts.TotalMemories,
		BundlesCount:  counts.TotalBundles,
		WebhooksCount: counts.WebhooksCount,
	}

	return &AnalyticsData{
		TenantID:             "global",
		AppID:                "",
		ExternalUserID:       "",
		TotalMemories:        counts.TotalMemories,
		TotalBundles:         counts.TotalBundles,
		MemoriesWithEmbeddings: counts.MemoriesWithEmbeddings,
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
