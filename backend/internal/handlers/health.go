package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/finops"
	"github.com/IlyesDjari/purp-tape/backend/internal/storage"
)

// HealthHandlers handles health checks and monitoring
type HealthHandlers struct {
	db  *db.Database
	r2  *storage.R2Client
	log *slog.Logger
}

// NewHealthHandlers creates health handler
func NewHealthHandlers(database *db.Database, r2Client *storage.R2Client, log *slog.Logger) *HealthHandlers {
	return &HealthHandlers{db: database, r2: r2Client, log: log}
}

// HealthStatus represents overall service health
type HealthStatus struct {
	Status       string                 `json:"status"` // healthy, degraded, unhealthy
	Timestamp    time.Time              `json:"timestamp"`
	Version      string                 `json:"version"`
	Uptime       int64                  `json:"uptime_seconds"`
	Components   map[string]ComponentStatus `json:"components"`
	Metrics      SystemMetrics          `json:"metrics"`
}

type ComponentStatus struct {
	Status    string `json:"status"` // healthy, degraded, unhealthy
	Latency   int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type SystemMetrics struct {
	Memory    MemoryMetrics    `json:"memory"`
	Goroutines int            `json:"goroutines"`
	Requests  RequestMetrics   `json:"requests"`
	Database  DatabaseMetrics  `json:"database"`
}

type MemoryMetrics struct {
	AllocMB      uint64 `json:"alloc_mb"`
	TotalAllocMB uint64 `json:"total_alloc_mb"`
	SysMB        uint64 `json:"sys_mb"`
	NumGC        uint32 `json:"num_gc"`
}

type RequestMetrics struct {
	Total        int64 `json:"total"`
	Errors       int64 `json:"errors"`
	ErrorRate    float64 `json:"error_rate"`
	AvgLatencyMS int64 `json:"avg_latency_ms"`
}

type DatabaseMetrics struct {
	OpenConnections int     `json:"open_connections"`
	InUseConnections int    `json:"in_use_connections"`
	IdleConnections int     `json:"idle_connections"`
	MaxConnections  int     `json:"max_connections"`
	PoolHealth      float64 `json:"pool_health_percent"` // 0-100
}

var (
	startTime     = time.Now()
	requestMetrics = &RequestMetrics{}
	metricsLock   sync.RWMutex
)

// GetHealth handles GET /health - basic liveness check
func (h *HealthHandlers) GetHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GetDeepHealth handles GET /health/deep - comprehensive health check
func (h *HealthHandlers) GetDeepHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	status := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Uptime:    int64(time.Since(startTime).Seconds()),
		Components: make(map[string]ComponentStatus),
	}

	// Check database
	dbStart := time.Now()
	dbErr := h.checkDatabase(ctx)
	dbLatency := time.Since(dbStart).Milliseconds()
	if dbErr != nil {
		status.Components["database"] = ComponentStatus{
			Status:    "unhealthy",
			Latency:   dbLatency,
			Error:     dbErr.Error(),
			Timestamp: time.Now(),
		}
		status.Status = "unhealthy"
	} else {
		status.Components["database"] = ComponentStatus{
			Status:    "healthy",
			Latency:   dbLatency,
			Timestamp: time.Now(),
		}
	}

	// Check R2 storage
	r2Start := time.Now()
	r2Err := h.checkR2(ctx)
	r2Latency := time.Since(r2Start).Milliseconds()
	if r2Err != nil {
		status.Components["storage"] = ComponentStatus{
			Status:    "degraded",
			Latency:   r2Latency,
			Error:     r2Err.Error(),
			Timestamp: time.Now(),
		}
		if status.Status == "healthy" {
			status.Status = "degraded"
		}
	} else {
		status.Components["storage"] = ComponentStatus{
			Status:    "healthy",
			Latency:   r2Latency,
			Timestamp: time.Now(),
		}
	}

	// Get system metrics
	status.Metrics = h.getSystemMetrics(ctx)

	finopsSettings := finops.LoadSettingsFromEnv()
	finopsStart := time.Now()
	finopsSnapshot, finopsErr := h.db.GetFinOpsSnapshot(ctx, finopsSettings.StorageCostPerGBMonth)
	finopsLatency := time.Since(finopsStart).Milliseconds()
	if finopsErr != nil {
		status.Components["finops"] = ComponentStatus{
			Status:    "degraded",
			Latency:   finopsLatency,
			Error:     finopsErr.Error(),
			Timestamp: time.Now(),
		}
		if status.Status == "healthy" {
			status.Status = "degraded"
		}
	} else {
		utilization := 0.0
		if finopsSettings.MonthlyBudgetUSD > 0 {
			utilization = finopsSnapshot.GoverningMonthlyCostUSD / finopsSettings.MonthlyBudgetUSD
		}

		componentStatus := "healthy"
		if utilization >= 1.0 || finopsSnapshot.FailedCleanupJobs > 0 {
			componentStatus = "degraded"
			if status.Status == "healthy" {
				status.Status = "degraded"
			}
		}

		status.Components["finops"] = ComponentStatus{
			Status:    componentStatus,
			Latency:   finopsLatency,
			Timestamp: time.Now(),
		}
	}

	// Return appropriate status code
	w.Header().Set("Content-Type", "application/json")
	if status.Status == "unhealthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(status)
}

// checkDatabase verifies database connectivity
func (h *HealthHandlers) checkDatabase(ctx context.Context) error {
	return h.db.Ping(ctx)
}

// checkR2 verifies R2 connectivity
func (h *HealthHandlers) checkR2(ctx context.Context) error {
	// Try to list objects (minimal operation)
	_, err := h.r2.ListFiles(ctx, "health-check", 1)
	return err
}

// getSystemMetrics collects system metrics
func (h *HealthHandlers) getSystemMetrics(ctx context.Context) SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metricsLock.RLock()
	errorRate := 0.0
	if requestMetrics.Total > 0 {
		errorRate = float64(requestMetrics.Errors) / float64(requestMetrics.Total)
	}
	metricsLock.RUnlock()

	dbStats := h.db.GetConnectionPoolStats()

	return SystemMetrics{
		Memory: MemoryMetrics{
			AllocMB:      m.Alloc / 1024 / 1024,
			TotalAllocMB: m.TotalAlloc / 1024 / 1024,
			SysMB:        m.Sys / 1024 / 1024,
			NumGC:        m.NumGC,
		},
		Goroutines: runtime.NumGoroutine(),
		Requests: RequestMetrics{
			Total:        requestMetrics.Total,
			Errors:       requestMetrics.Errors,
			ErrorRate:    errorRate,
			AvgLatencyMS: requestMetrics.AvgLatencyMS,
		},
		Database: DatabaseMetrics{
			OpenConnections:  dbStats.OpenConnections,
			InUseConnections: dbStats.InUse,
			IdleConnections:  dbStats.Idle,
			MaxConnections:   dbStats.MaxOpenConnections,
			PoolHealth:       float64(dbStats.OpenConnections) / float64(dbStats.MaxOpenConnections) * 100,
		},
	}
}

// RecordRequestMetric tracks request for metrics
func RecordRequestMetric(duration time.Duration, err error) {
	metricsLock.Lock()
	defer metricsLock.Unlock()

	requestMetrics.Total++
	if err != nil {
		requestMetrics.Errors++
	}

	// Rolling average
	oldTotal := requestMetrics.Total - 1
	if oldTotal > 0 {
		requestMetrics.AvgLatencyMS = (requestMetrics.AvgLatencyMS*oldTotal + duration.Milliseconds()) / requestMetrics.Total
	} else {
		requestMetrics.AvgLatencyMS = duration.Milliseconds()
	}
}

// GetReadiness handles GET /readiness - used by load balancers
func (h *HealthHandlers) GetReadiness(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Quick checks
	if err := h.db.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "not_ready",
			"reason": "database unavailable",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

// GetMetrics handles GET /metrics - Prometheus-compatible metrics
func (h *HealthHandlers) GetMetrics(w http.ResponseWriter, r *http.Request) {
	metricsLock.RLock()
	totalRequests := requestMetrics.Total
	totalErrors := requestMetrics.Errors
	metricsLock.RUnlock()

	finopsSettings := finops.LoadSettingsFromEnv()
	finopsSnapshot, finopsErr := h.db.GetFinOpsSnapshot(r.Context(), finopsSettings.StorageCostPerGBMonth)
	r2FinOps := h.r2.GetFinOpsMetrics()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Prometheus text format
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	
	fmt.Fprintf(w, "# HELP memory_alloc_bytes Current memory allocation in bytes\n")
	fmt.Fprintf(w, "# TYPE memory_alloc_bytes gauge\n")
	fmt.Fprintf(w, "memory_alloc_bytes %d\n\n", m.Alloc)

	fmt.Fprintf(w, "# HELP goroutines Current number of goroutines\n")
	fmt.Fprintf(w, "# TYPE goroutines gauge\n")
	fmt.Fprintf(w, "goroutines %d\n\n", runtime.NumGoroutine())

	fmt.Fprintf(w, "# HELP http_requests_total Total HTTP requests\n")
	fmt.Fprintf(w, "# TYPE http_requests_total counter\n")
	fmt.Fprintf(w, "http_requests_total %d\n\n", totalRequests)

	fmt.Fprintf(w, "# HELP http_errors_total Total HTTP errors\n")
	fmt.Fprintf(w, "# TYPE http_errors_total counter\n")
	fmt.Fprintf(w, "http_errors_total %d\n\n", totalErrors)

	fmt.Fprintf(w, "# HELP r2_presign_cache_hits_total Total presigned URL cache hits\n")
	fmt.Fprintf(w, "# TYPE r2_presign_cache_hits_total counter\n")
	fmt.Fprintf(w, "r2_presign_cache_hits_total %d\n\n", r2FinOps.PresignCacheHits)

	fmt.Fprintf(w, "# HELP r2_presign_cache_misses_total Total presigned URL cache misses\n")
	fmt.Fprintf(w, "# TYPE r2_presign_cache_misses_total counter\n")
	fmt.Fprintf(w, "r2_presign_cache_misses_total %d\n\n", r2FinOps.PresignCacheMisses)

	fmt.Fprintf(w, "# HELP r2_uploaded_bytes_total Total uploaded bytes to R2\n")
	fmt.Fprintf(w, "# TYPE r2_uploaded_bytes_total counter\n")
	fmt.Fprintf(w, "r2_uploaded_bytes_total %d\n\n", r2FinOps.UploadedBytes)

	fmt.Fprintf(w, "# HELP r2_deleted_objects_total Total deleted R2 objects\n")
	fmt.Fprintf(w, "# TYPE r2_deleted_objects_total counter\n")
	fmt.Fprintf(w, "r2_deleted_objects_total %d\n\n", r2FinOps.DeletedObjects)

	fmt.Fprintf(w, "# HELP r2_delete_operations_total Total single-object delete operations\n")
	fmt.Fprintf(w, "# TYPE r2_delete_operations_total counter\n")
	fmt.Fprintf(w, "r2_delete_operations_total %d\n\n", r2FinOps.DeleteOps)

	fmt.Fprintf(w, "# HELP r2_batch_delete_operations_total Total batch delete operations\n")
	fmt.Fprintf(w, "# TYPE r2_batch_delete_operations_total counter\n")
	fmt.Fprintf(w, "r2_batch_delete_operations_total %d\n\n", r2FinOps.BatchDeleteOps)

	fmt.Fprintf(w, "# HELP r2_lifecycle_policy_applied Indicates whether lifecycle policy is currently enforced (1=true)\n")
	fmt.Fprintf(w, "# TYPE r2_lifecycle_policy_applied gauge\n")
	fmt.Fprintf(w, "r2_lifecycle_policy_applied %d\n\n", r2FinOps.LifecyclePolicyApplied)

	if finopsErr == nil {
		budgetUtilization := 0.0
		if finopsSettings.MonthlyBudgetUSD > 0 {
			budgetUtilization = finopsSnapshot.GoverningMonthlyCostUSD / finopsSettings.MonthlyBudgetUSD
		}

		fmt.Fprintf(w, "# HELP finops_storage_active_bytes Total active storage bytes (tracks + images + offline)\n")
		fmt.Fprintf(w, "# TYPE finops_storage_active_bytes gauge\n")
		fmt.Fprintf(w, "finops_storage_active_bytes %d\n\n", finopsSnapshot.TotalActiveStorageBytes)

		fmt.Fprintf(w, "# HELP finops_storage_estimated_monthly_usd Estimated monthly storage cost in USD\n")
		fmt.Fprintf(w, "# TYPE finops_storage_estimated_monthly_usd gauge\n")
		fmt.Fprintf(w, "finops_storage_estimated_monthly_usd %.6f\n\n", finopsSnapshot.EstimatedMonthlyCostUSD)

		fmt.Fprintf(w, "# HELP finops_actual_monthly_usd Actual monthly cloud spend ingested via cost events\n")
		fmt.Fprintf(w, "# TYPE finops_actual_monthly_usd gauge\n")
		fmt.Fprintf(w, "finops_actual_monthly_usd %.6f\n\n", finopsSnapshot.ActualMonthlyCostUSD)

		fmt.Fprintf(w, "# HELP finops_governing_monthly_usd Governing monthly cost used for budget protection\n")
		fmt.Fprintf(w, "# TYPE finops_governing_monthly_usd gauge\n")
		fmt.Fprintf(w, "finops_governing_monthly_usd %.6f\n\n", finopsSnapshot.GoverningMonthlyCostUSD)

		fmt.Fprintf(w, "# HELP finops_budget_monthly_usd Configured monthly FinOps budget in USD\n")
		fmt.Fprintf(w, "# TYPE finops_budget_monthly_usd gauge\n")
		fmt.Fprintf(w, "finops_budget_monthly_usd %.6f\n\n", finopsSettings.MonthlyBudgetUSD)

		fmt.Fprintf(w, "# HELP finops_budget_utilization_ratio Governing monthly cost divided by budget\n")
		fmt.Fprintf(w, "# TYPE finops_budget_utilization_ratio gauge\n")
		fmt.Fprintf(w, "finops_budget_utilization_ratio %.6f\n\n", budgetUtilization)

		fmt.Fprintf(w, "# HELP finops_cleanup_pending_jobs Number of pending R2 cleanup jobs\n")
		fmt.Fprintf(w, "# TYPE finops_cleanup_pending_jobs gauge\n")
		fmt.Fprintf(w, "finops_cleanup_pending_jobs %d\n\n", finopsSnapshot.PendingCleanupJobs)

		fmt.Fprintf(w, "# HELP finops_cleanup_failed_jobs Number of failed R2 cleanup jobs\n")
		fmt.Fprintf(w, "# TYPE finops_cleanup_failed_jobs gauge\n")
		fmt.Fprintf(w, "finops_cleanup_failed_jobs %d\n\n", finopsSnapshot.FailedCleanupJobs)

		fmt.Fprintf(w, "# HELP finops_cleanup_pending_bytes Estimated pending cleanup bytes\n")
		fmt.Fprintf(w, "# TYPE finops_cleanup_pending_bytes gauge\n")
		fmt.Fprintf(w, "finops_cleanup_pending_bytes %d\n\n", finopsSnapshot.PendingCleanupBytes)
	}

	fmt.Fprintf(w, "# HELP uptime_seconds Service uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE uptime_seconds gauge\n")
	fmt.Fprintf(w, "uptime_seconds %d\n", int64(time.Since(startTime).Seconds()))
}
