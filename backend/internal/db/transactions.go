package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Tx wraps a database transaction for atomic operations.
type Tx struct {
	tx pgx.Tx
}

// WithTx executes a function within a transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
func (db *Database) WithTx(ctx context.Context, fn func(*Tx) error) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	dbTx := &Tx{tx: tx}

	// Execute the function
	if err := fn(dbTx); err != nil {
		// Rollback on error
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return fmt.Errorf("function failed: %w, rollback failed: %w", err, rollbackErr)
		}
		return err
	}

	// Commit if no error
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Exec executes a query within the transaction
func (t *Tx) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := t.tx.Exec(ctx, query, args...)
	return err
}

// QueryRow queries a single row within the transaction
func (t *Tx) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return t.tx.QueryRow(ctx, query, args...)
}

// Query queries multiple rows within the transaction
func (t *Tx) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return t.tx.Query(ctx, query, args...)
}

// BatchCreateProjects creates multiple projects in a single transaction.
func (db *Database) BatchCreateProjects(ctx context.Context, projects []interface{}) error {
	return db.WithTx(ctx, func(tx *Tx) error {
		query := `INSERT INTO projects (id, user_id, name, description, created_at, updated_at)
		         VALUES ($1, $2, $3, $4, $5, $6)`

		for _, p := range projects {
			if err := tx.Exec(ctx, query, p); err != nil {
				return fmt.Errorf("failed to insert project: %w", err)
			}
		}
		return nil
	})
}

// BatchDeleteTracks deletes multiple tracks in a single transaction (soft delete)
func (db *Database) BatchDeleteTracks(ctx context.Context, trackIDs []string, userID string) error {
	return db.WithTx(ctx, func(tx *Tx) error {
		query := `UPDATE tracks SET deleted_at = NOW() WHERE id = ANY($1) AND project_id IN (
		         SELECT id FROM projects WHERE user_id = $2 OR EXISTS (
		           SELECT 1 FROM project_shares WHERE project_id = projects.id AND shared_with_id = $2
		         ))`

		if err := tx.Exec(ctx, query, trackIDs, userID); err != nil {
			return fmt.Errorf("failed to delete tracks: %w", err)
		}
		return nil
	})
}
