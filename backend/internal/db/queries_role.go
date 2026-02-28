package db

import (
	"context"
	"database/sql"
)

// ============================================================================
// USER ROLE QUERIES
// ============================================================================

// IsFounder checks if a user has founder role (O(1) with index)
func (db *Database) IsFounder(ctx context.Context, userID string) (bool, error) {
	var role string
	err := db.pool.QueryRow(ctx, `SELECT role FROM users WHERE id = $1 AND deleted_at IS NULL`, userID).Scan(&role)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return role == "founder", nil
}

// IsAdmin checks if a user has admin or founder role
func (db *Database) IsAdmin(ctx context.Context, userID string) (bool, error) {
	var role string
	err := db.pool.QueryRow(ctx, `SELECT role FROM users WHERE id = $1 AND deleted_at IS NULL`, userID).Scan(&role)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return role == "admin" || role == "founder", nil
}

// GetUserRole retrieves user's role
func (db *Database) GetUserRole(ctx context.Context, userID string) (string, error) {
	var role string
	err := db.pool.QueryRow(ctx, `SELECT COALESCE(role, 'user') FROM users WHERE id = $1 AND deleted_at IS NULL`, userID).Scan(&role)
	if err == sql.ErrNoRows {
		return "user", nil
	}
	if err != nil {
		return "user", err
	}
	return role, nil
}

// SetUserRole sets a user's role (used for initialization)
func (db *Database) SetUserRole(ctx context.Context, userID, role string) error {
	_, err := db.pool.Exec(ctx, `UPDATE users SET role = $1 WHERE id = $2`, role, userID)
	return err
}

// FindUserByEmail retrieves a user by email (for initialization)
func (db *Database) FindUserByEmail(ctx context.Context, email string) (string, error) {
	var userID string
	err := db.pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL`, email).Scan(&userID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return userID, err
}

// InitializeFounderIfNeeded sets founder role based on environment variable
// Call this once on startup to promote founder user
func (db *Database) InitializeFounderIfNeeded(ctx context.Context, founderEmail string) error {
	if founderEmail == "" {
		return nil // No founding email configured
	}

	// Find user by email
	userID, err := db.FindUserByEmail(ctx, founderEmail)
	if err != nil {
		return err
	}
	if userID == "" {
		// User doesn't exist yet, that's OK - they'll be created at signup
		return nil
	}

	// Check if they already have founder/admin role
	role, err := db.GetUserRole(ctx, userID)
	if err != nil {
		return err
	}

	if role == "user" {
		// Promote to founder
		return db.SetUserRole(ctx, userID, "founder")
	}

	return nil
}
