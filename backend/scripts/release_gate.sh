#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PORT="${PORT:-18090}"
HOST="${HOST:-0.0.0.0}"
DB_URL="${DATABASE_URL:-postgres://purptape:devpassword123@localhost:5432/purptape?sslmode=disable}"

SERVER_LOG="$(mktemp)"
SERVER_PID=""

cleanup() {
	if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
		kill "${SERVER_PID}" >/dev/null 2>&1 || true
		wait "${SERVER_PID}" >/dev/null 2>&1 || true
	fi
	rm -f "${SERVER_LOG}"
}
trap cleanup EXIT

echo "[release-gate] starting API smoke server on ${HOST}:${PORT}"

/usr/bin/env -C "${ROOT_DIR}" \
	ENV="${ENV:-development}" \
	PORT="${PORT}" \
	HOST="${HOST}" \
	FRONTEND_URL="${FRONTEND_URL:-http://localhost:3000}" \
	DATABASE_URL="${DB_URL}" \
	SUPABASE_URL="${SUPABASE_URL:-http://localhost:54321}" \
	SUPABASE_ANON_KEY="${SUPABASE_ANON_KEY:-smoke-key}" \
	SUPABASE_SECRET_KEY="${SUPABASE_SECRET_KEY:-smoke-secret}" \
	R2_ACCESS_KEY_ID="${R2_ACCESS_KEY_ID:-smoke-r2-key}" \
	R2_SECRET_ACCESS_KEY="${R2_SECRET_ACCESS_KEY:-smoke-r2-secret}" \
	R2_ENDPOINT="${R2_ENDPOINT:-http://localhost:9000}" \
	R2_BUCKET_NAME="${R2_BUCKET_NAME:-smoke-bucket}" \
	R2_ACCOUNT_ID="${R2_ACCOUNT_ID:-smoke-account}" \
	OFFLINE_CLEANUP_SERVICE_TOKEN="${OFFLINE_CLEANUP_SERVICE_TOKEN:-smoke-token}" \
	go run ./cmd/api >"${SERVER_LOG}" 2>&1 &

SERVER_PID="$!"

for _ in $(seq 1 25); do
	if curl -fsS "http://localhost:${PORT}/health" >/dev/null 2>&1; then
		break
	fi
	if ! kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
		echo "[release-gate] server exited before health check"
		cat "${SERVER_LOG}"
		exit 1
	fi
	sleep 1
done

if ! curl -fsS "http://localhost:${PORT}/health" >/dev/null 2>&1; then
	echo "[release-gate] health endpoint failed"
	cat "${SERVER_LOG}"
	exit 1
fi

if ! curl -fsS "http://localhost:${PORT}/pricing/tiers" >/dev/null 2>&1; then
	echo "[release-gate] pricing endpoint failed"
	cat "${SERVER_LOG}"
	exit 1
fi

echo "[release-gate] startup probe passed"
