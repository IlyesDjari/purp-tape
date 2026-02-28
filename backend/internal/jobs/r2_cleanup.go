package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/storage"
)

// R2CleanupJob processes orphaned files from deleted projects and tracks
// R2CleanupJobRunner automatically cleans up R2 objects for soft-deleted items.
// Runs hourly, processes deletions older than 30 days to avoid accidental recovery issues
type R2CleanupJob struct {
	db     *db.Database
	r2     *storage.R2Client
	log    *slog.Logger
	ticker *time.Ticker
	stop   chan struct{}
}

// NewR2CleanupJob creates a new R2 cleanup job
func NewR2CleanupJob(database *db.Database, r2Client *storage.R2Client, log *slog.Logger) *R2CleanupJob {
	return &R2CleanupJob{
		db:     database,
		r2:     r2Client,
		log:    log,
		ticker: time.NewTicker(4 * time.Hour), // Run every 4 hours
		stop:   make(chan struct{}),
	}
}

// Start begins the cleanup job in background
func (rcj *R2CleanupJob) Start(ctx context.Context) {
	go func() {
		// Run cleanup immediately on startup
		rcj.processCleanup(ctx)

		// Then run on schedule
		for {
			select {
			case <-rcj.ticker.C:
				rcj.processCleanup(ctx)
			case <-rcj.stop:
				rcj.ticker.Stop()
				return
			case <-ctx.Done():
				rcj.ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the cleanup job
func (rcj *R2CleanupJob) Stop() {
	close(rcj.stop)
}

// processCleanup finds and deletes orphaned R2 files
func (rcj *R2CleanupJob) processCleanup(ctx context.Context) {
	rcj.log.Info("starting R2 cleanup job")

	// Find soft-deleted projects older than 30 days (safe to hard-delete from R2)
	// Gives users time to recover accidentally deleted projects from backups
	deletedProjects, err := rcj.db.GetSoftDeletedProjects(ctx, 30*24*time.Hour)
	if err != nil {
		rcj.log.Error("failed to get deleted projects", "error", err)
		return
	}

	deletedCount := 0
	errorCount := 0

	for _, proj := range deletedProjects {
		// Get all track versions for this project
		versions, err := rcj.db.GetAllProjectTrackVersions(ctx, proj.ID)
		if err != nil {
			rcj.log.Error("failed to get track versions for cleanup",
				"error", err,
				"project_id", proj.ID)
			errorCount++
			continue
		}

		// Delete from R2 in batches to reduce API calls and runtime
		const batchSize = 100
		batch := make([]string, 0, batchSize)

		flushBatch := func(keys []string) {
			if len(keys) == 0 {
				return
			}

			if err := rcj.r2.DeleteFilesBatch(ctx, keys); err != nil {
				rcj.log.Warn("failed to batch delete R2 files",
					"error", err,
					"count", len(keys),
					"project_id", proj.ID)
				for _, key := range keys {
					rcj.db.LogR2CleanupFailure(ctx, key, err.Error())
					errorCount++
				}
				return
			}

			deletedCount += len(keys)
		}

		for _, version := range versions {
			if version.R2ObjectKey == "" {
				continue
			}

			batch = append(batch, version.R2ObjectKey)
			if len(batch) >= batchSize {
				flushBatch(batch)
				batch = batch[:0]
			}
		}

		flushBatch(batch)

		// Mark project as hard-deleted in audit log
		if err := rcj.db.HardDeleteProject(ctx, proj.ID); err != nil {
			rcj.log.Warn("failed to hard-delete project after cleanup",
				"error", err,
				"project_id", proj.ID)
			errorCount++
		}
	}

	rcj.log.Info("R2 cleanup completed",
		"files_deleted", deletedCount,
		"errors", errorCount)
}
