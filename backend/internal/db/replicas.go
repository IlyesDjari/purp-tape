package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"
)

// ReadReplica configuration and management
type ReplicaPool struct {
	replicas []*sql.DB
	mu       sync.RWMutex
	log      *slog.Logger
}

// NewReplicaPool creates pool of read replicas
func NewReplicaPool(replicaURLs []string, log *slog.Logger) (*ReplicaPool, error) {
	if len(replicaURLs) == 0 {
		return nil, fmt.Errorf("at least one replica URL required")
	}

	pool := &ReplicaPool{
		replicas: make([]*sql.DB, 0, len(replicaURLs)),
		log:      log,
	}

	// Open connections to all replicas
	for i, url := range replicaURLs {
		db, err := sql.Open("postgres", url)
		if err != nil {
			return nil, fmt.Errorf("failed to open replica %d: %w", i, err)
		}

		// Configure connection pool
		db.SetMaxOpenConns(10)      // Smaller pool for replicas
		db.SetMaxIdleConns(3)
		db.SetConnMaxLifetime(5 * time.Minute)
		db.SetConnMaxIdleTime(10 * time.Minute)

		// Test connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := db.PingContext(ctx); err != nil {
			cancel()
			return nil, fmt.Errorf("replica %d ping failed: %w", i, err)
		}
		cancel()

		pool.replicas = append(pool.replicas, db)
		log.Info("replica connected", "url", url, "index", i)
	}

	return pool, nil
}

// GetReadConnection returns a replica connection (load balanced)
func (rp *ReplicaPool) GetReadConnection() *sql.DB {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	if len(rp.replicas) == 0 {
		return nil
	}

	// Simple round-robin or random selection
	idx := rand.Intn(len(rp.replicas))
	return rp.replicas[idx]
}

// QueryReplica executes SELECT on random replica
func (rp *ReplicaPool) QueryReplica(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	replica := rp.GetReadConnection()
	if replica == nil {
		return nil, fmt.Errorf("no replicas available")
	}

	return replica.QueryContext(ctx, query, args...)
}

// QueryRowReplica executes single row query on replica
func (rp *ReplicaPool) QueryRowReplica(ctx context.Context, query string, args ...interface{}) *sql.Row {
	replica := rp.GetReadConnection()
	if replica == nil {
		// This shouldn't happen in production
		rp.log.Error("no replicas available")
		return nil
	}

	return replica.QueryRowContext(ctx, query, args...)
}

// Health check replicas
func (rp *ReplicaPool) HealthCheck(ctx context.Context) map[int]error {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	errors := make(map[int]error)

	for i, replica := range rp.replicas {
		if err := replica.PingContext(ctx); err != nil {
			errors[i] = err
			rp.log.Warn("replica unhealthy", "index", i, "error", err)
		}
	}

	return errors
}

// Close closes all replica connections
func (rp *ReplicaPool) Close() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	for i, replica := range rp.replicas {
		if err := replica.Close(); err != nil {
			rp.log.Error("failed to close replica", "index", i, "error", err)
		}
	}

	return nil
}

// ============================================
// Integration with Database struct
// ============================================

// Enhanced Database struct with replication support
type DatabaseWithReplicas struct {
	primary  *sql.DB     // Write operations
	replicas *ReplicaPool // Read operations
	log      *slog.Logger
}

// NewDatabaseWithReplicas creates database with replica support
func NewDatabaseWithReplicas(primaryURL string, replicaURLs []string, log *slog.Logger) (*DatabaseWithReplicas, error) {
	// Connect to primary
	primary, err := sql.Open("postgres", primaryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to primary: %w", err)
	}

	// Configure primary
	primary.SetMaxOpenConns(25)
	primary.SetMaxIdleConns(5)
	primary.SetConnMaxLifetime(5 * time.Minute)
	primary.SetConnMaxIdleTime(10 * time.Minute)

	// Test primary
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := primary.PingContext(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("primary ping failed: %w", err)
	}
	cancel()

	log.Info("database primary connected", "url", primaryURL)

	// Create replica pool
	replicas, err := NewReplicaPool(replicaURLs, log)
	if err != nil {
		return nil, err
	}

	return &DatabaseWithReplicas{
		primary:  primary,
		replicas: replicas,
		log:      log,
	}, nil
}

// QueryPrimary - write operation goes to primary
func (dwr *DatabaseWithReplicas) QueryPrimary(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return dwr.primary.QueryContext(ctx, query, args...)
}

// QueryRowPrimary - single row from primary
func (dwr *DatabaseWithReplicas) QueryRowPrimary(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return dwr.primary.QueryRowContext(ctx, query, args...)
}

// ExecPrimary - INSERT/UPDATE/DELETE on primary
func (dwr *DatabaseWithReplicas) ExecPrimary(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return dwr.primary.ExecContext(ctx, query, args...)
}

// QueryReplica - read operation goes to replica
func (dwr *DatabaseWithReplicas) QueryReplica(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return dwr.replicas.QueryReplica(ctx, query, args...)
}

// QueryRowReplica - single row from replica
func (dwr *DatabaseWithReplicas) QueryRowReplica(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return dwr.replicas.QueryRowReplica(ctx, query, args...)
}

// HealthCheck checks primary and replicas
func (dwr *DatabaseWithReplicas) HealthCheck(ctx context.Context) map[string]interface{} {
	status := map[string]interface{}{
		"primary": nil,
		"replicas": make(map[string]interface{}),
	}

	// Check primary
	if err := dwr.primary.PingContext(ctx); err != nil {
		status["primary"] = err.Error()
		dwr.log.Error("primary unhealthy", "error", err)
	} else {
		status["primary"] = "healthy"
	}

	// Check replicas
	replicaErrors := dwr.replicas.HealthCheck(ctx)
	replicaStatus := status["replicas"].(map[string]interface{})
	for i, err := range replicaErrors {
		if err != nil {
			replicaStatus[fmt.Sprintf("replica_%d", i)] = err.Error()
		}
	}

	return status
}

// Close closes primary and replicas
func (dwr *DatabaseWithReplicas) Close() error {
	if err := dwr.primary.Close(); err != nil {
		dwr.log.Error("failed to close primary", "error", err)
	}

	if err := dwr.replicas.Close(); err != nil {
		dwr.log.Error("failed to close replicas", "error", err)
	}

	return nil
}

// Stats returns replica connection stats
func (dwr *DatabaseWithReplicas) ReplicaStats() {
	dwr.log.Info("primary stats",
		"open_conns", dwr.primary.Stats().OpenConnections,
		"in_use", dwr.primary.Stats().InUse,
		"idle", dwr.primary.Stats().Idle)
}
