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

// New creates a new database connection with connection pooling
func New(ctx context.Context, databaseURL string, maxConns, minConns int, log *slog.Logger) (*Database, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Optimize pool settings
	config.MaxConns = int32(maxConns)
	config.MinConns = int32(minConns)
	config.MaxConnIdleTime = 30 * time.Second  // Close idle connections faster
	config.MaxConnLifetime = 5 * time.Minute   // Recycle connections
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("database connection pool established",
		"max_conns", maxConns,
		"min_conns", minConns)

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
