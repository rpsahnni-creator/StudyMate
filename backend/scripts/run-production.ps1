# Production startup helper — edit values below before running.
# Usage: .\scripts\run-production.ps1

Set-Location (Split-Path $PSScriptRoot -Parent)

if (-not $env:ENVIRONMENT) { $env:ENVIRONMENT = "production" }
if (-not $env:PORT)         { $env:PORT = "8080" }

# ── Edit these before running ─────────────────────────────────────────────────
if (-not $env:DATABASE_URL) {
    $env:DATABASE_URL = "postgres://studyapp:CHANGE_ME@db.example.com:5432/studyapp?sslmode=require"
}
if (-not $env:VALKEY_ADDR) {
    $env:VALKEY_ADDR = "cache.example.com:6379"
}
if (-not $env:JWT_SECRET) {
    $env:JWT_SECRET = "CHANGE_ME_use_openssl_rand_base64_32"
}
if (-not $env:RAZORPAY_KEY_ID) {
    $env:RAZORPAY_KEY_ID = "rzp_live_CHANGE_ME"
}
# Optional:
# $env:RAZORPAY_WEBHOOK_SECRET = "whsec_CHANGE_ME"

# Local prod-like test with localhost Postgres (optional):
# $env:ALLOW_LOCALHOST_DB = "true"
# $env:DATABASE_URL = "postgres://localhost:5432/studyapp?sslmode=disable"

Write-Host "Starting StudyApp API (ENVIRONMENT=$($env:ENVIRONMENT), PORT=$($env:PORT))..."
go run cmd/api/main.go
