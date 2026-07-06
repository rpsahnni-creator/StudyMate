# Configure Resend for real OTP emails (free tier: 100/day).
# Get API key: https://resend.com/api-keys
Set-Location $PSScriptRoot\..

Write-Host ""
Write-Host "StudyMate - Resend email setup"
Write-Host ""
Write-Host "1. Sign up at https://resend.com (free)"
Write-Host "2. Create API key at https://resend.com/api-keys"
Write-Host "3. Paste key below (starts with re_)"
Write-Host ""
Write-Host "Note: onboarding@resend.dev only delivers to YOUR Resend account email"
Write-Host "      until you verify a custom domain."
Write-Host ""

$apiKey = Read-Host "Resend API key (re_...)"
$apiKey = $apiKey.Trim()
if (-not $apiKey.StartsWith("re_")) {
    Write-Error "Invalid key - must start with re_"
    exit 1
}

$from = 'StudyMate <onboarding@resend.dev>'

if (-not (Test-Path .env)) {
    Copy-Item .env.example .env
    Write-Host "Created .env from .env.example"
}

$lines = Get-Content .env
$keys = @{
    "EMAIL_PROVIDER" = "resend"
    "RESEND_API_KEY" = $apiKey
    "EMAIL_FROM"     = $from
}
$seen = @{}
$newLines = foreach ($line in $lines) {
    $matched = $false
    foreach ($key in $keys.Keys) {
        if ($line -match "^\s*$key=") {
            $seen[$key] = $true
            "$key=$($keys[$key])"
            $matched = $true
            break
        }
    }
    if (-not $matched) { $line }
}
foreach ($key in $keys.Keys) {
    if (-not $seen[$key]) { $newLines += "$key=$($keys[$key])" }
}

$utf8NoBom = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllLines((Join-Path (Get-Location) ".env"), $newLines, $utf8NoBom)

Write-Host ""
Write-Host "Updated .env:"
Write-Host "  EMAIL_PROVIDER=resend"
Write-Host "  EMAIL_FROM=$from"
Write-Host "  RESEND_API_KEY=***"
Write-Host ""
Write-Host "Restart backend: go run .\cmd\api"
Write-Host "Then register - OTP will arrive in Gmail (Resend test rules apply)."
