#!/usr/bin/env bash
# Production startup helper — edit values below before running.
# Usage: ./scripts/run-production.sh

set -euo pipefail

cd "$(dirname "$0")/.."

export ENVIRONMENT="${ENVIRONMENT:-production}"
export PORT="${PORT:-8080}"

# ── Edit these before running ─────────────────────────────────────────────────
export DATABASE_URL="${DATABASE_URL:-postgres://studyapp:CHANGE_ME@db.example.com:5432/studyapp?sslmode=require}"
export VALKEY_ADDR="${VALKEY_ADDR:-cache.example.com:6379}"
export JWT_SECRET="${JWT_SECRET:-CHANGE_ME_use_openssl_rand_base64_32}"
export RAZORPAY_KEY_ID="${RAZORPAY_KEY_ID:-rzp_live_CHANGE_ME}"
export RAZORPAY_WEBHOOK_SECRET="${RAZORPAY_WEBHOOK_SECRET:-}"

# Local prod-like test with localhost Postgres (optional):
# export ALLOW_LOCALHOST_DB=true
# export DATABASE_URL="postgres://localhost:5432/studyapp?sslmode=disable"

echo "Starting StudyApp API (ENVIRONMENT=$ENVIRONMENT, PORT=$PORT)..."
exec go run cmd/api/main.go
