package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

type ConnectionPoolStats struct {
	OpenConnections    int
	InUse              int
	Idle               int
	MaxOpenConnections int
}

// New creates a new database connection with aggressive optimization for high-throughput audio app
// Tuned for: RLS policies (read-heavy), streaming (write-moderate), analytics (batch)
func New(ctx context.Context, databaseURL string, maxConns, minConns int, log *slog.Logger) (*Database, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// PERFORMANCE: Aggressive pooling for high-throughput scenarios
	// 2x larger default pool for concurrent RLS checks
	if maxConns <= 0 {
		maxConns = 40 // Default: handle bursts up to 40 concurrent queries
	}
	if minConns <= 0 {
		minConns = 10 // Keep 10 warm connections
	}

	config.MaxConns = int32(maxConns)
	config.MinConns = int32(minConns)
	
	// Connection lifecycle tuning
	config.MaxConnIdleTime = 20 * time.Second   // Aggressive idle timeout (reduce stale connections)
	config.MaxConnLifetime = 3 * time.Minute    // Cycle connections frequently (avoid long-lived transaction leaks)
	config.HealthCheckPeriod = 5 * time.Second  // Check pool health every 5s
	
	// Query optimization
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement // Cache prepared statements
	
	// Network optimization
	config.ConnConfig.ConnectTimeout = 5 * time.Second
	config.ConnConfig.RuntimeParams = map[string]string{
		"application_name":         "purptape-api",
		"jit":                      "off",           // Disable JIT for consistent performance
		"random_page_cost":         "1.1",           // Assume SSD storage
		"effective_cache_size":     "2GB",           // Tune planner
		"shared_buffers":           "256MB",         // From connection pooling
		"work_mem":                 "8MB",           // Per operation
		"maintenance_work_mem":     "64MB",          // For index creation
		"max_parallel_workers_per_gather": "2",     // Parallel query execution
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("database connection pool established with high-performance tuning",
		"max_conns", maxConns,
		"min_conns", minConns,
		"max_conn_idle_time_sec", 20,
		"max_conn_lifetime_sec", 180,
		"health_check_period_sec", 5)

	return &Database{pool: pool, log: log}, nil
}

// Close closes the connection pool
func (db *Database) Close() {
	db.pool.Close()
	db.log.Info("database connection pool closed")
}

// Pool returns the underlying pgxpool.Pool for advanced usage
func (db *Database) Pool() *pgxpool.Pool {
	return db.pool
}

func (db *Database) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

func (db *Database) GetConnectionPoolStats() ConnectionPoolStats {
	stats := db.pool.Stat()
	return ConnectionPoolStats{
		OpenConnections:    int(stats.TotalConns()),
		InUse:              int(stats.AcquiredConns()),
		Idle:               int(stats.IdleConns()),
		MaxOpenConnections: int(stats.MaxConns()),
	}
}

// GetPerformanceMetrics returns database performance metrics for monitoring
// Use in health checks or metrics endpoints
func (db *Database) GetPerformanceMetrics(ctx context.Context) map[string]interface{} {
	stats := db.GetConnectionPoolStats()
	
	// Calculate key metrics
	utilizationPercent := 0
	if stats.MaxOpenConnections > 0 {
		utilizationPercent = (stats.InUse * 100) / stats.MaxOpenConnections
	}

	return map[string]interface{}{
		"connection_pool": map[string]interface{}{
			"open_connections":     stats.OpenConnections,
			"in_use":              stats.InUse,
			"idle":                stats.Idle,
			"max_connections":     stats.MaxOpenConnections,
			"utilization_percent": utilizationPercent,
			"available_capacity":  stats.Idle,
		},
		"performance_status": map[string]string{
			"health": getPoolHealthStatus(utilizationPercent),
		},
	}
}

func getPoolHealthStatus(utilizationPercent int) string {
	if utilizationPercent < 50 {
		return "healthy"
	} else if utilizationPercent < 80 {
		return "good"
	} else if utilizationPercent < 95 {
		return "warning"
	}
	return "critical"
}
