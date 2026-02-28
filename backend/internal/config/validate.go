package config

import (
	"fmt"
)

// Validate validates all required configuration [MEDIUM: Configuration validation]
func (c *Config) Validate() error {
	// Server
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid PORT: must be 1-65535, got %d", c.Port)
	}

	// Database
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.DBMaxConns < 1 {
		return fmt.Errorf("DB_MAX_CONNS must be >= 1, got %d", c.DBMaxConns)
	}
	if c.DBMinConns < 1 {
		return fmt.Errorf("DB_MIN_CONNS must be >= 1, got %d", c.DBMinConns)
	}
	if c.DBMaxConns < c.DBMinConns {
		return fmt.Errorf("DB_MAX_CONNS must be >= DB_MIN_CONNS")
	}

	// Supabase
	if c.SupabaseURL == "" {
		return fmt.Errorf("SUPABASE_URL is required")
	}
	if c.SupabaseAnonKey == "" {
		return fmt.Errorf("SUPABASE_ANON_KEY is required")
	}
	if c.SupabaseSecretKey == "" {
		return fmt.Errorf("SUPABASE_SECRET_KEY is required")
	}

	// R2
	if c.R2AccessKeyID == "" || c.R2SecretAccessKey == "" {
		return fmt.Errorf("R2_ACCESS_KEY_ID and R2_SECRET_ACCESS_KEY are required")
	}
	if c.R2BucketName == "" {
		return fmt.Errorf("R2_BUCKET_NAME is required")
	}
	if c.R2Endpoint == "" {
		return fmt.Errorf("R2_ENDPOINT is required")
	}

	return nil
}

// IsProduction returns true if running in production
func (c *Config) IsProduction() bool {
	return c.Env == "production" || c.Env == "prod"
}

// IsDevelopment returns true if running in development
func (c *Config) IsDevelopment() bool {
	return c.Env == "development" || c.Env == "dev" || c.Env == ""
}
