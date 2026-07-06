#Requires -Version 5.1
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Root = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$Backend = Join-Path $Root "backend"
$ToolsDir = Join-Path $Backend ".tools\minio"
$DataDir = Join-Path $Backend ".tools\minio-data"
$MinioExe = Join-Path $ToolsDir "minio.exe"
$PidFile = Join-Path $ToolsDir "minio.pid"

$MinioUser = if ($env:MINIO_ROOT_USER) { $env:MINIO_ROOT_USER } else { "minioadmin" }
$MinioPass = if ($env:MINIO_ROOT_PASSWORD) { $env:MINIO_ROOT_PASSWORD } else { "minioadmin123" }
$Bucket = "studyapp-temp"

function Test-MinioHealthy {
    try {
        $r = Invoke-WebRequest -Uri "http://localhost:9000/minio/health/live" -UseBasicParsing -TimeoutSec 3
        return $r.StatusCode -eq 200
    } catch {
        return $false
    }
}

function Start-MinioDocker {
    Write-Host "Starting MinIO via Docker Compose..."
    Push-Location $Root
    try {
        docker compose up -d minio minio-init
        if ($LASTEXITCODE -ne 0) { throw "docker compose failed" }
    } finally {
        Pop-Location
    }
}

function Ensure-MinioBinary {
    New-Item -ItemType Directory -Force -Path $ToolsDir, $DataDir | Out-Null
    if (-not (Test-Path $MinioExe)) {
        Write-Host "Downloading MinIO for Windows..."
        $url = "https://dl.min.io/server/minio/release/windows-amd64/minio.exe"
        Invoke-WebRequest -Uri $url -OutFile $MinioExe
    }
}

function Start-MinioNative {
    Ensure-MinioBinary

    if (Test-Path $PidFile) {
        $oldPid = Get-Content $PidFile -ErrorAction SilentlyContinue
        if ($oldPid -and (Get-Process -Id $oldPid -ErrorAction SilentlyContinue)) {
            Write-Host "MinIO already running (PID $oldPid)."
            return
        }
    }

    Write-Host "Starting MinIO native process..."
    $env:MINIO_ROOT_USER = $MinioUser
    $env:MINIO_ROOT_PASSWORD = $MinioPass

    $proc = Start-Process -FilePath $MinioExe `
        -ArgumentList @("server", $DataDir, "--console-address", ":9001") `
        -PassThru -WindowStyle Hidden

    $proc.Id | Set-Content $PidFile
    Write-Host "MinIO started (PID $($proc.Id))."
}

function Wait-MinioReady {
    Write-Host "Waiting for MinIO health..."
    for ($i = 1; $i -le 30; $i++) {
        if (Test-MinioHealthy) {
            Write-Host "MinIO is healthy."
            return
        }
        Start-Sleep -Seconds 1
    }
    throw "MinIO did not become healthy within 30s. Check port 9000 is free."
}

if (Test-MinioHealthy) {
    Write-Host "MinIO already running at http://localhost:9000"
} elseif (Get-Command docker -ErrorAction SilentlyContinue) {
    Start-MinioDocker
    Wait-MinioReady
} else {
    Write-Host "Docker not found - using native MinIO binary."
    Start-MinioNative
    Wait-MinioReady
}

Write-Host ""
Write-Host "MinIO ready:"
Write-Host "  API:     http://localhost:9000"
Write-Host "  Console: http://localhost:9001"
Write-Host "  Bucket:  $Bucket (auto-created when backend starts)"
Write-Host ""
Write-Host "Restart backend:"
Write-Host "  cd E:\study-app\backend"
Write-Host "  go run ./cmd/api"
