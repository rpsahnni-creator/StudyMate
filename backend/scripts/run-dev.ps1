# Local backend dev server — loads backend/.env automatically via config.Load().
Set-Location $PSScriptRoot\..

Write-Host "Starting StudyApp backend on http://localhost:8080 ..."
go run ./cmd/api
