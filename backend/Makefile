# Build targets
.PHONY: build dev docker-build docker-up docker-down test coverage lint migration-lint migration-check quality-gate release-gate s-tier-gate

# Build the API binary
build:
	go build -o bin/purptape-api ./cmd/api

# Run locally (requires PostgreSQL and environment variables)
dev:
	go run ./cmd/api

# Run with Docker Compose
docker-up:
	docker compose up --build

# Stop Docker containers
docker-down:
	docker compose down

# Rebuild Docker image
docker-build:
	docker compose build

# Run tests
test:
	go test -v ./...

# Generate test coverage
coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Lint code (requires golangci-lint)
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...

# Run database migrations (local)
migrate-up:
	@echo "Run these commands to migrate locally:"
	@echo "psql -h localhost -U purptape -d purptape < migrations/001_create_users_table.sql"
	@echo "psql -h localhost -U purptape -d purptape < migrations/002_create_projects_table.sql"
	@echo "psql -h localhost -U purptape -d purptape < migrations/003_create_tracks_table.sql"
	@echo "psql -h localhost -U purptape -d purptape < migrations/004_create_track_versions_table.sql"
	@echo "psql -h localhost -U purptape -d purptape < migrations/005_create_project_shares_table.sql"
	@echo "psql -h localhost -U purptape -d purptape < migrations/006_create_audit_logs_table.sql"

# Replays all migrations against a running local PostgreSQL and fails on first error
migration-lint:
	MIGRATIONS_DIR=migrations ./scripts/check_migration_sequence.sh

migration-check:
	DB_HOST=localhost DB_PORT=5432 DB_USER=purptape DB_PASSWORD=devpassword123 DB_NAME=purptape MIGRATIONS_DIR=migrations ./scripts/check_migrations.sh

# Main local quality gate: migration replay + full backend compile/tests
quality-gate: migration-check test

# Full local release gate including static analysis and startup probe
release-gate: quality-gate
	go vet ./...
	./scripts/release_gate.sh

# Strict production-readiness gate (fails on dirty workspace, missing env, failed replay/tests/smoke)
s-tier-gate:
	./scripts/s_tier_gate.sh

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Setup development environment
setup: 
	cp .env.example .env
	@echo "Created .env file. Please fill in your credentials."
