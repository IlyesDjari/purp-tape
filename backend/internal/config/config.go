package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Server
	Port       int
	Host       string
	Env        string
	FrontendURL string

	// Database
	DatabaseURL        string
	DBMaxConns         int
	DBMinConns         int
	DBMaxIdleTime      time.Duration
	DBConnMaxLifetime  time.Duration

	// Supabase Auth
	SupabaseURL        string
	SupabaseAnonKey    string
	SupabaseSecretKey  string

	// Cloudflare R2
	R2AccessKeyID      string
	R2SecretAccessKey  string
	R2Endpoint         string
	R2BucketName       string
	R2AccountID        string

	// JWT
	JWTSecret string

	// CORS
	CORSAllowedOrigins []string

	// Security [MEDIUM FIX]
	PresignedURLExpiry    time.Duration // Presigned URL validity period
	EncryptionKeyBase64   string         // Base64-encoded encryption key for sensitive data
	ShareTokenExpiry      time.Duration  // Share link token validity period
	RateLimitRequests     int            // Requests per window
	RateLimitWindow       time.Duration  // Rate limit window
	JobWorkerConcurrency  int            // Number of concurrent background workers per instance
	JobBatchSize          int            // Number of jobs claimed per poll

	// FinOps
	FinOpsStorageCostPerGBMonth float64 // Cost assumption for R2 storage ($/GB/month)
	FinOpsMonthlyBudgetUSD      float64 // Budget threshold for monthly storage cost
	FinOpsBudgetGuardEnabled    bool    // Enables budget guard for expensive jobs
	FinOpsUploadBlockEnabled    bool    // Blocks new uploads when projected cost exceeds threshold
	FinOpsBudgetGuardRatio      float64 // Ratio of budget where guard kicks in (1.0 = 100%)
	FinOpsEnforceR2Lifecycle    bool    // Ensures bucket lifecycle policy is applied on startup
	FinOpsR2LifecycleStrict     bool    // Fail startup if lifecycle policy cannot be applied
	FinOpsCostIngestToken       string  // Shared secret for FinOps cost ingestion endpoint
}

func Load() (*Config, error) {
	env := getEnv("ENV", "development")
	
	// Optimize connection pool based on environment
	// Production: larger pool for high concurrency
	// Development: small pool to avoid resource waste
	maxConns := 5
	minConns := 1
	if env == "production" {
		maxConns = getEnvInt("DB_MAX_CONNS", 25)
		minConns = getEnvInt("DB_MIN_CONNS", 5)
	} else {
		maxConns = getEnvInt("DB_MAX_CONNS", 5)
		minConns = getEnvInt("DB_MIN_CONNS", 1)
	}

	cfg := &Config{
		Port:               getEnvInt("PORT", 8080),
		Host:               getEnv("HOST", "0.0.0.0"),
		Env:                env,
		FrontendURL:        getEnv("FRONTEND_URL", "https://purptape.com"),
		DBMaxConns:         maxConns,
		DBMinConns:         minConns,
		DBMaxIdleTime:      30 * time.Second,
		DBConnMaxLifetime:  5 * time.Minute,
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		SupabaseURL:        getEnv("SUPABASE_URL", ""),
		SupabaseAnonKey:    getEnv("SUPABASE_ANON_KEY", ""),
		SupabaseSecretKey:  getEnv("SUPABASE_SECRET_KEY", ""),
		R2AccessKeyID:      getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretAccessKey:  getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2Endpoint:         getEnv("R2_ENDPOINT", ""),
		R2BucketName:       getEnv("R2_BUCKET_NAME", ""),
		R2AccountID:        getEnv("R2_ACCOUNT_ID", ""),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		// [COST OPTIMIZATION] Presigned URLs: shorter expiry = fewer API calls to refresh
		PresignedURLExpiry:  time.Duration(getEnvInt("PRESIGNED_URL_EXPIRY_MINUTES", 3)) * time.Minute,
		EncryptionKeyBase64: getEnv("ENCRYPTION_KEY_BASE64", ""),
		ShareTokenExpiry:    time.Duration(getEnvInt("SHARE_TOKEN_EXPIRY_DAYS", 30)) * 24 * time.Hour,
		// [COST OPTIMIZATION] Rate limiting: balanced for abuse prevention without blocking legitimate users
		RateLimitRequests:   getEnvInt("RATE_LIMIT_REQUESTS", 150),
		RateLimitWindow:     time.Duration(getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second,
		JobWorkerConcurrency: getEnvInt("JOB_WORKER_CONCURRENCY", 4),
		JobBatchSize:         getEnvInt("JOB_BATCH_SIZE", 32),
		FinOpsStorageCostPerGBMonth: getEnvFloat("FINOPS_STORAGE_COST_PER_GB_MONTH", 0.015),
		FinOpsMonthlyBudgetUSD:      getEnvFloat("FINOPS_MONTHLY_BUDGET_USD", 25.0),
		FinOpsBudgetGuardEnabled:    getEnvBool("FINOPS_BUDGET_GUARD_ENABLED", false),
		FinOpsUploadBlockEnabled:    getEnvBool("FINOPS_UPLOAD_BLOCK_ENABLED", false),
		FinOpsBudgetGuardRatio:      getEnvFloat("FINOPS_BUDGET_GUARD_RATIO", 1.0),
		FinOpsEnforceR2Lifecycle:    getEnvBool("FINOPS_ENFORCE_R2_LIFECYCLE", false),
		FinOpsR2LifecycleStrict:     getEnvBool("FINOPS_R2_LIFECYCLE_STRICT", false),
		FinOpsCostIngestToken:       getEnv("FINOPS_COST_INGEST_TOKEN", ""),
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.SupabaseURL == "" || cfg.SupabaseAnonKey == "" || cfg.SupabaseSecretKey == "" {
		return nil, fmt.Errorf("SUPABASE_URL, SUPABASE_ANON_KEY, and SUPABASE_SECRET_KEY are required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intVal, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intVal
}

func getEnvFloat(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	floatVal, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return floatVal
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
