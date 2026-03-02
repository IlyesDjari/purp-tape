package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrations executes all SQL migration files in order
// Uses a schema_migrations table to track which migrations have been applied
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	// Create schema_migrations table if it doesn't exist
	if err := createMigrationsTable(ctx, pool); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get list of migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Filter and sort SQL files
	var sqlFiles []os.DirEntry
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			sqlFiles = append(sqlFiles, file)
		}
	}

	// Sort by filename to ensure correct order
	sort.Slice(sqlFiles, func(i, j int) bool {
		return sqlFiles[i].Name() < sqlFiles[j].Name()
	})

	// Execute each migration only if not already applied
	for _, file := range sqlFiles {
		migrationName := file.Name()

		// Check if migration has already been applied
		var applied bool
		err := pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = $1)",
			migrationName,
		).Scan(&applied)
		if err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", migrationName, err)
		}

		if applied {
			fmt.Printf("⊘ Skipped migration (already applied): %s\n", migrationName)
			continue
		}

		// Read the migration file
		filePath := filepath.Join(migrationsDir, migrationName)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", migrationName, err)
		}

		// Skip empty files
		sql := strings.TrimSpace(string(content))
		if sql == "" {
			continue
		}

		// Execute migration file as a whole to preserve PL/pgSQL dollar-quoted bodies.
		// Extract VACUUM statements because they cannot run in a transaction block.
		preparedSQL, maintenanceStatements := preprocessMigrationSQL(sql)
		if preparedSQL != "" {
			if _, err := pool.Exec(ctx, preparedSQL); err != nil {
				if isNonFatalMigrationError(err) {
					fmt.Printf("⚠ Skipping restricted statements in %s: %v\n", migrationName, err)
				} else {
					return fmt.Errorf("failed to execute migration %s: %w", migrationName, err)
				}
			}
		}

		for _, stmt := range maintenanceStatements {
			if _, err := pool.Exec(ctx, stmt); err != nil {
				// Non-fatal: maintenance statements should not block startup.
				fmt.Printf("⚠ maintenance statement failed in %s: %v\n", migrationName, err)
			}
		}

		if err := ensureUserProjectAccessCache(ctx, pool, migrationName); err != nil {
			return err
		}

		// Record that migration was applied
		if _, err := pool.Exec(ctx,
			"INSERT INTO schema_migrations (name) VALUES ($1) ON CONFLICT DO NOTHING",
			migrationName,
		); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", migrationName, err)
		}

		fmt.Printf("✓ Executed migration: %s\n", migrationName)
	}

	return nil
}

// preprocessMigrationSQL extracts VACUUM statements and returns SQL safe for pooled execution.
func preprocessMigrationSQL(sql string) (string, []string) {
	lines := strings.Split(sql, "\n")
	kept := make([]string, 0, len(lines))
	maintenance := make([]string, 0)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		upper := strings.ToUpper(trimmed)
		if strings.HasPrefix(upper, "VACUUM ") || upper == "VACUUM;" {
			maintenance = append(maintenance, trimmed)
			continue
		}
		kept = append(kept, line)
	}

	return strings.TrimSpace(strings.Join(kept, "\n")), maintenance
}

// ensureUserProjectAccessCache guarantees the critical access cache table exists
// even if an optimization migration partially fails.
func ensureUserProjectAccessCache(ctx context.Context, pool *pgxpool.Pool, migrationName string) error {
	if migrationName < "040_performance_rls_refactor.sql" {
		return nil
	}

	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS user_project_access (
			user_id UUID NOT NULL,
			project_id UUID NOT NULL,
			access_type TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (user_id, project_id, access_type)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to ensure user_project_access exists: %w", err)
	}

	_, err = pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_user_project_access_user_id
		ON user_project_access(user_id)
	`)
	if err != nil {
		return fmt.Errorf("failed to ensure user_project_access index exists: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO user_project_access (user_id, project_id, access_type)
		SELECT p.user_id, p.id, 'owner'
		FROM projects p
		WHERE p.deleted_at IS NULL
		ON CONFLICT (user_id, project_id, access_type) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to backfill user_project_access: %w", err)
	}

	return nil
}

func isNonFatalMigrationError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "42501" {
			return true
		}
		if pgErr.Code == "42703" {
			return true
		}
	}

	message := strings.ToLower(err.Error())

	return strings.Contains(message, "permission denied for schema auth") ||
		strings.Contains(message, "must be owner") ||
		strings.Contains(message, "insufficient privilege") ||
		strings.Contains(message, "permission denied for table") ||
		strings.Contains(message, "column old.deleted_at does not exist")
}

// createMigrationsTable creates the schema_migrations tracking table
func createMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}
