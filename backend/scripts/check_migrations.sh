#!/usr/bin/env bash
set -euo pipefail

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-purptape}"
DB_PASSWORD="${DB_PASSWORD:-devpassword123}"
DB_NAME="${DB_NAME:-purptape}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-migrations}"
DB_CONTAINER="${DB_CONTAINER:-purptape-postgres}"

export PGPASSWORD="$DB_PASSWORD"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/check_migration_sequence.sh"

use_docker_exec=false
if ! command -v psql >/dev/null 2>&1 || ! command -v pg_isready >/dev/null 2>&1; then
  if command -v docker >/dev/null 2>&1; then
    use_docker_exec=true
  else
    echo "Neither local psql/pg_isready nor docker is available" >&2
    exit 1
  fi
fi

if [[ "$use_docker_exec" == "false" ]]; then
  echo "Waiting for PostgreSQL at ${DB_HOST}:${DB_PORT}..."
  for _ in $(seq 1 60); do
    if pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done

  if ! pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" >/dev/null 2>&1; then
    echo "PostgreSQL is not ready" >&2
    exit 1
  fi
else
  echo "Using docker exec mode against container ${DB_CONTAINER}"
  for _ in $(seq 1 60); do
    if docker exec "$DB_CONTAINER" pg_isready -U "$DB_USER" >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done

  if ! docker exec "$DB_CONTAINER" pg_isready -U "$DB_USER" >/dev/null 2>&1; then
    echo "PostgreSQL container is not ready" >&2
    exit 1
  fi
fi

echo "Resetting database schema to clean state"
if [[ "$use_docker_exec" == "false" ]]; then
  psql \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    -v ON_ERROR_STOP=1 \
    -c 'DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;' >/dev/null
else
  docker exec -i "$DB_CONTAINER" env PGPASSWORD="$DB_PASSWORD" psql \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    -v ON_ERROR_STOP=1 \
    -c 'DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;' >/dev/null
fi

echo "Applying migrations from ${MIGRATIONS_DIR}"
for migration in "$MIGRATIONS_DIR"/*.sql; do
  echo "-> ${migration}"
  if [[ "$use_docker_exec" == "false" ]]; then
    psql \
      -h "$DB_HOST" \
      -p "$DB_PORT" \
      -U "$DB_USER" \
      -d "$DB_NAME" \
      -v ON_ERROR_STOP=1 \
      -f "$migration" >/dev/null
  else
    docker exec -i "$DB_CONTAINER" env PGPASSWORD="$DB_PASSWORD" psql \
      -U "$DB_USER" \
      -d "$DB_NAME" \
      -v ON_ERROR_STOP=1 \
      -f - >/dev/null < "$migration"
  fi

done

echo "Migration replay completed successfully"
