#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

ALLOW_DIRTY="${ALLOW_DIRTY:-0}"
SKIP_MIGRATION_REPLAY="${SKIP_MIGRATION_REPLAY:-0}"

fail() {
  echo "[s-tier-gate] $1" >&2
  exit 1
}

echo "[s-tier-gate] starting strict production-readiness gate"

if [[ "${ALLOW_DIRTY}" != "1" ]]; then
  if [[ -n "$(git status --porcelain)" ]]; then
    fail "workspace is dirty; commit/stash changes or run with ALLOW_DIRTY=1"
  fi
fi

echo "[s-tier-gate] workspace cleanliness check passed"

required_vars=(
  DATABASE_URL
  SUPABASE_URL
  SUPABASE_ANON_KEY
  SUPABASE_SECRET_KEY
  R2_ACCESS_KEY_ID
  R2_SECRET_ACCESS_KEY
  R2_ENDPOINT
  R2_BUCKET_NAME
  R2_ACCOUNT_ID
  FRONTEND_URL
)

for var_name in "${required_vars[@]}"; do
  if [[ -z "${!var_name:-}" ]]; then
    fail "missing required env var: ${var_name}"
  fi
done

echo "[s-tier-gate] required env var check passed"

if [[ "${SKIP_MIGRATION_REPLAY}" != "1" ]]; then
  DB_HOST="${DB_HOST:-localhost}" \
  DB_PORT="${DB_PORT:-5432}" \
  DB_USER="${DB_USER:-purptape}" \
  DB_PASSWORD="${DB_PASSWORD:-devpassword123}" \
  DB_NAME="${DB_NAME:-purptape}" \
  MIGRATIONS_DIR="migrations" \
  ./scripts/check_migrations.sh
else
  echo "[s-tier-gate] skipping migration replay (SKIP_MIGRATION_REPLAY=1)"
fi

GOFLAGS="${GOFLAGS:-}" go test ./...
go vet ./...

PORT="${PORT:-18090}" ./scripts/release_gate.sh

echo "[s-tier-gate] PASS"
