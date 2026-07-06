# StudyApp Backend — Production Startup Guide

Server startup se pehle `config.Validate()` chalti hai. Production mein insecure defaults par server **start nahi hoga**.

---

## Config validation summary

| Environment | Default `JWT_SECRET` | `localhost` in `DATABASE_URL` | Missing `RAZORPAY_KEY_ID` | Result |
|-------------|----------------------|----------------------------------|---------------------------|--------|
| `development` | Allowed | Allowed | Allowed | Starts + **warning logs** |
| `staging` | **Blocked** | **Blocked** (unless override) | **Blocked** | Must pass all production checks |
| `production` | **Blocked** | **Blocked** (unless override) | **Blocked** | Must pass all production checks |

### Production checks (sab ek saath report hote hain)

| Variable | Rule |
|----------|------|
| `ENVIRONMENT` | Must be set (`production`) |
| `PORT` | Non-empty |
| `DATABASE_URL` | Non-empty; no `localhost` unless `ALLOW_LOCALHOST_DB=true` |
| `VALKEY_ADDR` | Non-empty |
| `JWT_SECRET` | Non-empty; must **not** be `dev-secret-change-in-production` |
| `RAZORPAY_KEY_ID` | Non-empty |
| `ALLOWED_ORIGINS` | Non-empty; no localhost origins |
| `FRONTEND_URL` | Non-empty |

Billing validation (`billing.Config.Validate()`) additionally requires in production/staging:

| Variable | Rule |
|----------|------|
| `RAZORPAY_KEY_SECRET` | Non-empty |
| `RAZORPAY_WEBHOOK_SECRET` | Non-empty |

### Development warnings (server start hota hai)

```
config warning: JWT_SECRET not set, using insecure default (ok for local dev only)
config warning: DATABASE_URL not set, using default "postgres://localhost:5432/..."
...
```

---

## Pre-flight checklist

Before first production start:

- [ ] PostgreSQL running; all migrations applied (`migrations/*.sql` in order)
- [ ] Valkey / Redis reachable from app host
- [ ] Strong `JWT_SECRET` generated (see below)
- [ ] `DATABASE_URL` points to managed Postgres (not localhost)
- [ ] `RAZORPAY_KEY_ID` set (live key for production)
- [ ] `RAZORPAY_KEY_SECRET` set (server-side order creation)
- [ ] `RAZORPAY_WEBHOOK_SECRET` set and webhook URL configured in Razorpay dashboard
- [ ] `ALLOWED_ORIGINS` and `FRONTEND_URL` set for your deployed frontend
- [ ] First admin: set `ADMIN_BOOTSTRAP_EMAIL`, register that user, restart API, re-login
- [ ] Frontend `NEXT_PUBLIC_API_URL` and mobile `EXPO_PUBLIC_API_URL` point to this API

### Generate a strong JWT secret

```bash
# Linux / macOS
openssl rand -base64 32

# Windows PowerShell
[Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }))
```

---

## Option A — Environment variables (recommended)

### Linux / macOS

```bash
cd backend

export ENVIRONMENT=production
export PORT=8080
export DATABASE_URL="postgres://studyapp:YOUR_PASSWORD@db.example.com:5432/studyapp?sslmode=require"
export VALKEY_ADDR="cache.example.com:6379"
export JWT_SECRET="YOUR_OPENSSL_RAND_BASE64_32_OUTPUT"
export RAZORPAY_KEY_ID="rzp_live_xxxxxxxx"
export RAZORPAY_WEBHOOK_SECRET="whsec_xxxxxxxx"   # optional but recommended

go run cmd/api/main.go
```

One-liner:

```bash
ENVIRONMENT=production \
PORT=8080 \
DATABASE_URL="postgres://studyapp:YOUR_PASSWORD@db.example.com:5432/studyapp?sslmode=require" \
VALKEY_ADDR="cache.example.com:6379" \
JWT_SECRET="YOUR_STRONG_SECRET" \
RAZORPAY_KEY_ID="rzp_live_xxxxxxxx" \
go run cmd/api/main.go
```

### Windows PowerShell

```powershell
cd backend

$env:ENVIRONMENT = "production"
$env:PORT = "8080"
$env:DATABASE_URL = "postgres://studyapp:YOUR_PASSWORD@db.example.com:5432/studyapp?sslmode=require"
$env:VALKEY_ADDR = "cache.example.com:6379"
$env:JWT_SECRET = "YOUR_OPENSSL_RAND_BASE64_32_OUTPUT"
$env:RAZORPAY_KEY_ID = "rzp_live_xxxxxxxx"
$env:RAZORPAY_WEBHOOK_SECRET = "whsec_xxxxxxxx"

go run cmd/api/main.go
```

### Helper scripts

```bash
# Linux / macOS — edit variables inside script first
./scripts/run-production.sh

# Windows PowerShell — edit variables inside script first
.\scripts\run-production.ps1
```

---

## Option B — `.env` file (local prod-like testing only)

1. Copy template: `cp .env.example .env`
2. Fill in production values in `.env`
3. Load before start:

```bash
# Linux / macOS (requires export from .env)
set -a && source .env && set +a && go run cmd/api/main.go
```

```powershell
# Windows — set vars manually from .env or use run-production.ps1
Get-Content .env | ForEach-Object {
  if ($_ -match '^([^#=]+)=(.*)$') { Set-Item -Path "env:$($matches[1])" -Value $matches[2] }
}
go run cmd/api/main.go
```

**Never commit `.env` with real secrets.** `.env` is gitignored.

---

## Option C — Build binary for deployment

```bash
cd backend

export ENVIRONMENT=production
# ... set all required env vars ...

go build -o studyapp-api ./cmd/api
./studyapp-api
```

Docker / systemd / Kubernetes: inject the same env vars into the container or unit file.

---

## Verify server started

```bash
curl http://localhost:8080/health
# Expected: ok - database and cache both reachable
```

---

## Common startup failures

| Error message | Fix |
|---------------|-----|
| `JWT_SECRET must not use the default dev secret in production` | Set a unique secret; never use `dev-secret-change-in-production` |
| `DATABASE_URL must not use localhost in production` | Use managed Postgres host, or set `ALLOW_LOCALHOST_DB=true` for local prod testing only |
| `RAZORPAY_KEY_ID is required in production` | Set live/test Razorpay key ID |
| `DATABASE_URL is required` (+ other fields) | All required vars missing — error lists every field at once |
| `failed to connect to postgres` | DB unreachable or wrong credentials (passed Validate, connection failed) |
| `failed to connect to cache` | Valkey/Redis not running or wrong `VALKEY_ADDR` |

---

## Local development (no extra config)

```bash
cd backend
go run cmd/api/main.go
```

Uses defaults. You'll see config warnings — that is expected and safe for local dev.

---

## Related client env vars

| App | Variable | Production example |
|-----|----------|-------------------|
| Frontend | `NEXT_PUBLIC_API_URL` | `https://api.studyapp.example.com` |
| Mobile (EAS) | `EXPO_PUBLIC_API_URL` | `https://api.studyapp.example.com` |
