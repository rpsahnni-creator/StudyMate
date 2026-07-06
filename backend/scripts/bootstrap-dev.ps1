# Local dev bootstrap helper
Set-Location $PSScriptRoot\..

Write-Host "StudyApp dev bootstrap"
Write-Host "  1. Ensure Postgres + Valkey are running"
Write-Host "  2. Set ADMIN_BOOTSTRAP_EMAIL in .env (default: kijitechnology@gmail.com)"
Write-Host "  3. Register that email via /auth/register, then restart the API"
Write-Host "  4. Re-login to get admin JWT; career_goals_module auto-enabled in development"
Write-Host ""

if (-not (Test-Path .env)) {
    Copy-Item .env.example .env
    Write-Host "Created .env from .env.example"
}

go run migrations/run_migrations.go
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "Starting API on http://localhost:8080 ..."
go run ./cmd/api
