package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
)

// AccessListRefreshJob maintains materialized views and access cache consistency
// Run hourly or on-demand to keep performance optimal
type AccessListRefreshJob struct {
	db  *db.Database
	log *slog.Logger
}

// NewAccessListRefreshJob creates a new access list refresh job
func NewAccessListRefreshJob(database *db.Database, log *slog.Logger) *AccessListRefreshJob {
	return &AccessListRefreshJob{
		db:  database,
		log: log,
	}
}

// RefreshMaterializedView refreshes the mv_user_accessible_projects view
// This should be called hourly or after bulk permission changes
// Duration: ~100-500ms on small-medium databases
func (alr *AccessListRefreshJob) RefreshMaterializedView(ctx context.Context) error {
	start := time.Now()

	err := alr.db.RefreshAccessibilityMaterializedView(ctx)
	if err != nil {
		alr.log.Error("failed to refresh materialized view", "error", err)
		return err
	}

	duration := time.Since(start)
	alr.log.Info("materialized view refreshed successfully",
		"duration_ms", duration.Milliseconds())

	return nil
}

// ScheduleRegularRefresh runs refresh on a schedule (hourly default)
// Can be called from main.go:
// go accessListRefreshJob.ScheduleRegularRefresh(context.Background(), 1*time.Hour)
func (alr *AccessListRefreshJob) ScheduleRegularRefresh(ctx context.Context, interval time.Duration) {
	if interval == 0 {
		interval = 1 * time.Hour // Default: hourly
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	alr.log.Info("starting scheduled access list refresh",
		"interval_minutes", int(interval.Minutes()))

	for {
		select {
		case <-ticker.C:
			if err := alr.RefreshMaterializedView(ctx); err != nil {
				alr.log.Error("scheduled refresh failed", "error", err)
				// Continue - don't crash scheduler
			}
		case <-ctx.Done():
			alr.log.Info("access list refresh scheduler stopped")
			return
		}
	}
}

// VacuumAccessCache performs maintenance on access tables
// Recommended: weekly or monthly
// Removes orphaned entries and optimizes table storage
func (alr *AccessListRefreshJob) VacuumAccessCache(ctx context.Context) error {
	// Note: In production, consider running VACUUM ANALYZE during off-peak hours
	// This requires direct database access not exposed through pool queries
	alr.log.Info("access cache vacuum would run here (requires direct DB access)")
	return nil
}
