# Local dev bootstrap helper
Set-Location $PSScriptRoot\..

Write-Host "StudyApp dev bootstrap"
Write-Host "  1. Ensure Postgres + Valkey are running"
Write-Host "  2. Set ADMIN_BOOTSTRAP_EMAIL in .env (default: kijitechnology@gmail.com)"
Write-Host "  3. Register that email via /auth/register, then restart the API"
Write-Host "  4. Re-login to get admin JWT; career_goals_module auto-enabled in development"
Write-Host ""

$lanIp = (Get-NetIPAddress -AddressFamily IPv4 | Where-Object {
    $_.IPAddress -notlike "127.*" -and $_.PrefixOrigin -ne "WellKnown"
} | Select-Object -First 1).IPAddress

if ($lanIp) {
    Write-Host "Phone ke liye mobile/.env mein set karein:"
    Write-Host "  EXPO_PUBLIC_API_URL=http://${lanIp}:8080"
    Write-Host ""
}

$fwRule = "StudyApp Backend 8080"
$existing = netsh advfirewall firewall show rule name="$fwRule" 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "Adding Windows Firewall rule for TCP 8080 (phone uploads)..."
    netsh advfirewall firewall add rule name="$fwRule" dir=in action=allow protocol=TCP localport=8080 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Firewall rule added."
    } else {
        Write-Host "Could not add firewall rule — run PowerShell as Administrator if phone cannot connect."
    }
}

if (-not (Test-Path .env)) {
    Copy-Item .env.example .env
    Write-Host "Created .env from .env.example"
}

go run migrations/run_migrations.go
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "Starting API on http://localhost:8080 (LAN: http://${lanIp}:8080) ..."
go run ./cmd/api
