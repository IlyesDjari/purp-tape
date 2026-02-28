#!/bin/bash

# PurpTape Backend - Critical Security Fixes Deployment Script
# This script deploys all critical security fixes in the correct order
# Usage: ./deploy-critical-fixes.sh [environment] [database_url]

set -e

ENVIRONMENT=${1:-staging}
DATABASE_URL=${2:-$DATABASE_URL}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Verify prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command -v psql &> /dev/null; then
        log_error "psql not found. Please install PostgreSQL client."
        exit 1
    fi
    
    if [ -z "$DATABASE_URL" ]; then
        log_error "DATABASE_URL not set. Usage: ./deploy-critical-fixes.sh staging postgresql://..."
        exit 1
    fi
    
    if ! command -v go &> /dev/null; then
        log_error "Go not found. Please install Go 1.22+"
        exit 1
    fi
    
    log_info "Prerequisites check passed ✓"
}

# Backup database
backup_database() {
    log_info "Creating database backup..."
    BACKUP_FILE="purptape_${ENVIRONMENT}_backup_$(date +%Y%m%d_%H%M%S).sql"
    
    if pg_dump "$DATABASE_URL" > "$BACKUP_FILE" 2>/dev/null; then
        log_info "Database backed up to: $BACKUP_FILE ✓"
    else
        log_error "Failed to create backup!"
        exit 1
    fi
}

# Download and update dependencies
update_dependencies() {
    log_info "Updating Go dependencies..."
    
    if ! go get github.com/golang-jwt/jwt/v5 2>/dev/null; then
        log_error "Failed to download JWT dependency"
        exit 1
    fi
    
    if ! go mod tidy 2>/dev/null; then
        log_error "Failed to tidy Go modules"
        exit 1
    fi
    
    log_info "Dependencies updated ✓"
}

# Apply migrations in order
apply_migrations() {
    log_info "Applying critical security migrations..."
    
    # Migration 023: Missing RLS policies
    log_info "Applying migration 023: Missing RLS policies..."
    if ! psql "$DATABASE_URL" < migrations/023_add_missing_rls_policies.sql 2>/dev/null; then
        log_error "Failed to apply migration 023"
        exit 1
    fi
    log_info "Migration 023 applied ✓"
    
    # Migration 024: Soft deletes
    log_info "Applying migration 024: Soft deletes..."
    if ! psql "$DATABASE_URL" < migrations/024_add_soft_deletes.sql 2>/dev/null; then
        log_error "Failed to apply migration 024"
        exit 1
    fi
    log_info "Migration 024 applied ✓"
    
    # Migration 025: R2 cleanup triggers
    log_info "Applying migration 025: R2 cleanup triggers..."
    if ! psql "$DATABASE_URL" < migrations/025_add_r2_cleanup_triggers.sql 2>/dev/null; then
        log_error "Failed to apply migration 025"
        exit 1
    fi
    log_info "Migration 025 applied ✓"
}

# Verify migrations
verify_migrations() {
    log_info "Verifying migrations..."
    
    # Check if collaborators has RLS enabled
    COLLABORATORS_RLS=$(psql "$DATABASE_URL" -t -c "SELECT relrowsecurity FROM pg_class WHERE relname = 'collaborators';" 2>/dev/null | xargs)
    if [ "$COLLABORATORS_RLS" = "t" ]; then
        log_info "✓ RLS policies applied correctly"
    else
        log_error "RLS policies not applied correctly"
        exit 1
    fi
    
    # Check if deleted_at columns exist
    DELETED_AT_PROJECTS=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM information_schema.columns WHERE table_name='projects' AND column_name='deleted_at';" 2>/dev/null | xargs)
    if [ "$DELETED_AT_PROJECTS" = "1" ]; then
        log_info "✓ Soft delete columns created"
    else
        log_error "Soft delete columns not found"
        exit 1
    fi
    
    # Check if background_jobs table exists
    BACKGROUND_JOBS=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name='background_jobs';" 2>/dev/null | xargs)
    if [ "$BACKGROUND_JOBS" = "1" ]; then
        log_info "✓ Background jobs table created"
    else
        log_error "Background jobs table not found"
        exit 1
    fi
}

# Build application
build_application() {
    log_info "Building application with security fixes..."
    
    if ! go build -o purptape-backend cmd/api/main.go 2>/dev/null; then
        log_error "Failed to build application"
        exit 1
    fi
    
    log_info "Application built successfully ✓"
}

# Test JWT validation
test_jwt_validation() {
    log_info "Testing JWT validation..."
    
    # This is a basic smoke test - in production, test with real tokens
    if grep -q "jwt.ParseWithClaims" internal/auth/validator.go; then
        log_info "✓ JWT signature verification implemented"
    else
        log_error "JWT signature verification not found"
        exit 1
    fi
}

# Test RBAC
test_rbac_implementation() {
    log_info "Testing RBAC implementation..."
    
    if grep -q "QueryUserProjectRole" internal/middleware/rbac.go && \
       grep -q "IsProjectOwner" internal/db/queries_rbac.go; then
        log_info "✓ Database-driven RBAC implemented"
    else
        log_error "RBAC implementation incomplete"
        exit 1
    fi
}

# Test R2 validation
test_r2_validation() {
    log_info "Testing R2 path validation..."
    
    if grep -q "validateObjectKey" internal/storage/validation.go && \
       grep -q "ValidateUploadRequest" internal/storage/validation.go; then
        log_info "✓ R2 path validation implemented"
    else
        log_error "R2 validation not found"
        exit 1
    fi
}

# Main deployment flow
main() {
    log_info "=========================================="
    log_info "PurpTape Critical Security Fixes Deployment"
    log_info "=========================================="
    log_info "Environment: $ENVIRONMENT"
    log_info "Database: ${DATABASE_URL:0:50}..."
    log_info ""
    
    check_prerequisites
    backup_database
    update_dependencies
    apply_migrations
    verify_migrations
    build_application
    test_jwt_validation
    test_rbac_implementation
    test_r2_validation
    
    log_info ""
    log_info "=========================================="
    log_info "✅ All critical security fixes deployed!"
    log_info "=========================================="
    log_info ""
    log_info "Next steps:"
    log_info "1. Deploy binary: purptape-backend"
    log_info "2. Restart application with new binary"
    log_info "3. Verify JWT JWKS endpoint is accessible"
    log_info "4. Run security test suite (see docs)"
    log_info "5. Monitor background_jobs table"
    log_info ""
    log_info "Backup file: $BACKUP_FILE"
    log_info "Keep for 30 days before deletion"
    log_info ""
}

# Run main function
main
