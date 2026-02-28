package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/IlyesDjari/purp-tape/backend/internal/auth"
	"github.com/IlyesDjari/purp-tape/backend/internal/config"
	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/jobs"
	"github.com/IlyesDjari/purp-tape/backend/internal/middleware"
	"github.com/IlyesDjari/purp-tape/backend/internal/notifications"
	"github.com/IlyesDjari/purp-tape/backend/internal/storage"
)

func main() {
	// Load environment variables from .env
	godotenv.Load()

	// Setup logging with JSON output
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize database
	database, err := db.New(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns, log)
	if err != nil {
		log.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Initialize R2 storage client
	r2Client, err := storage.NewR2Client(
		cfg.R2AccessKeyID,
		cfg.R2SecretAccessKey,
		cfg.R2Endpoint,
		cfg.R2BucketName,
		cfg.R2AccountID,
		log,
	)
	if err != nil {
		log.Error("failed to initialize R2 client", "error", err)
		os.Exit(1)
	}

	if cfg.FinOpsEnforceR2Lifecycle {
		if err := r2Client.EnsureLifecyclePolicies(context.Background()); err != nil {
			if cfg.FinOpsR2LifecycleStrict {
				log.Error("failed to enforce R2 lifecycle policies in strict mode", "error", err)
				os.Exit(1)
			}
			log.Warn("failed to enforce R2 lifecycle policies", "error", err)
		}
	}

	// Initialize auth validator
	authValidator := auth.NewValidator(cfg.SupabaseURL, cfg.SupabaseAnonKey, cfg.SupabaseSecretKey)

	// Initialize notification services
	pushNotifSvc := notifications.NewPushNotificationService(database, cfg.FCMServerKey, log)
	prefsSvc := notifications.NewPreferencesService(database, log)
	notifSvc := notifications.NewNotificationService(database, pushNotifSvc, prefsSvc, log)

	appHandlers := newAppHandlers(database, r2Client, notifSvc, pushNotifSvc, prefsSvc, log)
	jobProcessor := jobs.NewJobProcessor(database, r2Client, log, cfg.JobWorkerConcurrency, cfg.JobBatchSize)
	jobCtx, stopJobs := context.WithCancel(context.Background())
	defer stopJobs()
	go jobProcessor.ProcessPendingJobs(jobCtx)

	// Create router using Go 1.22 ServeMux with path patterns
	mux := http.NewServeMux()

	// Middleware chain with explicit CORS origin allow-list
	corsOrigins := []string{cfg.FrontendURL}
	if cfg.Env == "development" {
		corsOrigins = append(corsOrigins, "http://localhost:3000", "http://localhost:5173")
	}

	rateLimiter := middleware.NewConfigurableRateLimiter(cfg, log)
	handler := middleware.Chain(
		mux,
		middleware.RecoveryMiddleware(log),
		middleware.RequestMetricsMiddleware(log),
		middleware.GzipMiddleware(),
		rateLimiter.RateLimitMiddleware,
		middleware.CORSMiddleware(corsOrigins),
		middleware.LoggingMiddleware(log),
	)
	withAuth := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.AuthMiddleware(authValidator, log)(next).ServeHTTP
	}

	registerRoutes(mux, appHandlers, withAuth)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Info("starting server", "addr", addr, "env", cfg.Env)

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal (SIGINT, SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("shutting down server")
	stopJobs()

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	log.Info("server stopped")
}
