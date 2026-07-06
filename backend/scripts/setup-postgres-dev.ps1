# One-time local Postgres setup for StudyApp dev (Windows).
# Prompts for your PostgreSQL "postgres" user password (set during install).
Set-Location $PSScriptRoot\..

$psql = "C:\Program Files\PostgreSQL\18\bin\psql.exe"
if (-not (Test-Path $psql)) {
    $found = Get-ChildItem "C:\Program Files\PostgreSQL" -Recurse -Filter psql.exe -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($found) { $psql = $found.FullName } else { Write-Error "psql.exe not found. Install PostgreSQL or add it to PATH."; exit 1 }
}

Write-Host ""
Write-Host "StudyApp — local Postgres setup"
Write-Host "  User: postgres"
Write-Host "  Database: studyapp"
Write-Host ""
Write-Host "Enter the password you chose when installing PostgreSQL 18:"
$secure = Read-Host -AsSecureString
$bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
$pgPassword = [Runtime.InteropServices.Marshal]::PtrToStringAuto($bstr)
[Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)

$env:PGPASSWORD = $pgPassword
& $psql -U postgres -h localhost -d postgres -v ON_ERROR_STOP=1 -c "SELECT 1" 2>$null | Out-Null
if ($LASTEXITCODE -ne 0) {
    Remove-Item Env:PGPASSWORD -ErrorAction SilentlyContinue
    Write-Error "Could not connect as postgres. Check password and that PostgreSQL service is running."
    exit 1
}

$dbExists = & $psql -U postgres -h localhost -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='studyapp'"
if ($dbExists -ne "1") {
    Write-Host "Creating database studyapp..."
    & $psql -U postgres -h localhost -d postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE studyapp"
    if ($LASTEXITCODE -ne 0) { Remove-Item Env:PGPASSWORD -ErrorAction SilentlyContinue; exit 1 }
} else {
    Write-Host "Database studyapp already exists."
}

Remove-Item Env:PGPASSWORD -ErrorAction SilentlyContinue

# URL-encode password for connection string (basic: @ : / etc.)
$encoded = [uri]::EscapeDataString($pgPassword)
$dbUrl = "postgres://postgres:${encoded}@localhost:5432/studyapp?sslmode=disable"

if (-not (Test-Path .env)) {
    Copy-Item .env.example .env
    Write-Host "Created .env from .env.example"
}

$lines = Get-Content .env
$updated = $false
$newLines = foreach ($line in $lines) {
    if ($line -match '^\s*DATABASE_URL=') {
        $updated = $true
        "DATABASE_URL=$dbUrl"
    } else {
        $line
    }
}
if (-not $updated) { $newLines += "DATABASE_URL=$dbUrl" }
Set-Content -Path .env -Value $newLines -Encoding utf8NoBOM

Write-Host "Updated DATABASE_URL in .env"
Write-Host "Running migrations..."
$env:DATABASE_URL = $dbUrl
go run migrations/run_migrations.go
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host ""
Write-Host "Done. Start the API with: .\scripts\run-dev.ps1"
