package cleanup

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"cortex/internal/models"
	"cortex/internal/store"
)

// Config holds cleanup job configuration.
type Config struct {
	// DryRun if true, no writes are performed (only counts).
	DryRun bool
	// ArchiveByExpiry: archive memories with expires_at <= now
	ArchiveByExpiry bool
	// DeleteArchivedOlderThan: if > 0, permanently delete archived memories older than this duration (e.g. 720h = 30 days)
	DeleteArchivedOlderThan time.Duration
	// MergeSimilar: run merge of similar memories
	MergeSimilar bool
	// MergeMinSimilarity threshold (0–1), e.g. 0.95
	MergeMinSimilarity float64
	// MergeMaxPairs per run (0 = no limit)
	MergeMaxPairs int
	// ArchiveLowImportance: archive active memories with importance below this (e.g. 2)
	ArchiveLowImportance bool
	// LowImportanceThreshold (1–10)
	LowImportanceThreshold int
}

// Stats holds cleanup run statistics.
type Stats struct {
	ArchivedByExpiry   int64
	DeletedArchived   int64
	MergedPairs       int64
	ArchivedLowImport int64
}

// DefaultConfig returns a conservative default config (only TTL archive enabled).
func DefaultConfig() Config {
	return Config{
		ArchiveByExpiry:         true,
		DeleteArchivedOlderThan: 0,
		MergeSimilar:            false,
		MergeMinSimilarity:      0.95,
		MergeMaxPairs:           50,
		ArchiveLowImportance:    false,
		LowImportanceThreshold:  2,
	}
}

// ConfigFromEnv returns Config from environment variables.
// CORTEX_CLEANUP_DRY_RUN=true, CORTEX_CLEANUP_ARCHIVE_EXPIRY=true,
// CORTEX_CLEANUP_DELETE_ARCHIVED_AFTER=720h (30d), CORTEX_CLEANUP_MERGE_SIMILAR=false,
// CORTEX_CLEANUP_MERGE_SIMILARITY=0.95, CORTEX_CLEANUP_MERGE_MAX_PAIRS=50,
// CORTEX_CLEANUP_ARCHIVE_LOW_IMPORTANCE=false, CORTEX_CLEANUP_LOW_IMPORTANCE_THRESHOLD=2
func ConfigFromEnv() Config {
	c := DefaultConfig()
	if v := os.Getenv("CORTEX_CLEANUP_DRY_RUN"); v == "true" || v == "1" {
		c.DryRun = true
	}
	if v := os.Getenv("CORTEX_CLEANUP_ARCHIVE_EXPIRY"); v == "false" || v == "0" {
		c.ArchiveByExpiry = false
	}
	if v := os.Getenv("CORTEX_CLEANUP_DELETE_ARCHIVED_AFTER"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.DeleteArchivedOlderThan = d
		}
	}
	if v := os.Getenv("CORTEX_CLEANUP_MERGE_SIMILAR"); v == "true" || v == "1" {
		c.MergeSimilar = true
	}
	if v := os.Getenv("CORTEX_CLEANUP_MERGE_SIMILARITY"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 && f <= 1 {
			c.MergeMinSimilarity = f
		}
	}
	if v := os.Getenv("CORTEX_CLEANUP_MERGE_MAX_PAIRS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			c.MergeMaxPairs = n
		}
	}
	if v := os.Getenv("CORTEX_CLEANUP_ARCHIVE_LOW_IMPORTANCE"); v == "true" || v == "1" {
		c.ArchiveLowImportance = true
	}
	if v := os.Getenv("CORTEX_CLEANUP_LOW_IMPORTANCE_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 10 {
			c.LowImportanceThreshold = n
		}
	}
	return c
}

// RunCleanup runs one cleanup pass: TTL archive, optional delete archived, optional merge similar, optional archive low-importance.
func RunCleanup(ctx context.Context, s *store.CortexStore, cfg Config) (Stats, error) {
	var stats Stats
	now := time.Now()

	if cfg.ArchiveByExpiry && !cfg.DryRun {
		n, err := s.ArchiveMemoriesByExpiry(now)
		if err != nil {
			return stats, err
		}
		stats.ArchivedByExpiry = n
		if n > 0 {
			slog.Info("cleanup: archived by expiry", "count", n)
		}
	} else if cfg.ArchiveByExpiry && cfg.DryRun {
		// Count only: would need a separate store method; skip for dry run or do a raw count
		_ = now
	}

	if cfg.DeleteArchivedOlderThan > 0 && !cfg.DryRun {
		cutoff := now.Add(-cfg.DeleteArchivedOlderThan)
		n, err := s.DeleteArchivedOlderThan(cutoff)
		if err != nil {
			return stats, err
		}
		stats.DeletedArchived = n
		if n > 0 {
			slog.Info("cleanup: deleted archived older than", "count", n, "cutoff", cutoff)
		}
	}

	if cfg.MergeSimilar && !cfg.DryRun {
		// We need to iterate tenants; for simplicity we run merge per (app_id, external_user_id) from memories
		// Get distinct tenants from memories (active only)
		var tenants []struct {
			AppID          string
			ExternalUserID string
		}
		if err := s.GetDB().Model(&models.Memory{}).
			Where("status = ?", models.MemoryStatusActive).
			Distinct("app_id", "external_user_id").
			Find(&tenants).Error; err != nil {
			return stats, err
		}
		limit := cfg.MergeMaxPairs
		if limit <= 0 {
			limit = 100
		}
		for _, t := range tenants {
			pairs, err := s.FindSimilarMemoryPairs(t.AppID, t.ExternalUserID, nil, cfg.MergeMinSimilarity, limit)
			if err != nil {
				slog.Warn("cleanup: find similar failed", "appId", t.AppID, "userId", t.ExternalUserID, "error", err)
				continue
			}
			for _, p := range pairs {
				if err := s.MergeMemories(p[0], p[1], t.AppID, t.ExternalUserID); err != nil {
					slog.Warn("cleanup: merge failed", "keep", p[0], "merge", p[1], "error", err)
					continue
				}
				stats.MergedPairs++
			}
		}
		if stats.MergedPairs > 0 {
			slog.Info("cleanup: merged similar", "pairs", stats.MergedPairs)
		}
	}

	if cfg.ArchiveLowImportance && !cfg.DryRun {
		thresh := cfg.LowImportanceThreshold
		if thresh < 1 {
			thresh = 1
		}
		if thresh > 10 {
			thresh = 10
		}
		res := s.GetDB().Model(&models.Memory{}).
			Where("status = ? AND importance < ?", models.MemoryStatusActive, thresh).
			Update("status", models.MemoryStatusArchived)
		if res.Error != nil {
			return stats, res.Error
		}
		stats.ArchivedLowImport = res.RowsAffected
		if stats.ArchivedLowImport > 0 {
			slog.Info("cleanup: archived low importance", "count", stats.ArchivedLowImport, "threshold", thresh)
		}
	}

	return stats, nil
}

// StartCleanupTicker runs RunCleanup every interval (from CORTEX_CLEANUP_INTERVAL, default 24h).
// It blocks until ctx is cancelled. Call in a goroutine.
func StartCleanupTicker(ctx context.Context, s *store.CortexStore, interval time.Duration) {
	if interval <= 0 {
		if v := os.Getenv("CORTEX_CLEANUP_INTERVAL"); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				interval = d
			}
		}
		if interval <= 0 {
			interval = 24 * time.Hour
		}
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	cfg := ConfigFromEnv()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := RunCleanup(ctx, s, cfg)
			if err != nil {
				slog.Error("cleanup ticker failed", "error", err)
			}
		}
	}
}
