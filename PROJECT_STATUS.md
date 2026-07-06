# StudyApp — Final Project Status

> **Reference document** · Last updated: **2026-07-05**  
> Read-only status snapshot. Ye file project ka single source of truth hai — tech stack, structure, progress, aur workflows.

---

## 1. Executive Summary

**StudyApp** ek educational monorepo hai jisme students NCERT / state-board material scan karke AI-generated quizzes banate hain. Teen independently deployable clients ek shared Go backend se connect hote hain.

| Metric | Value |
|--------|-------|
| **Overall completion** | ~**55%** (foundation strong, core product flow partial) |
| **Production-ready MVP** | ❌ Not yet |
| **Current phase** | Phase 1 — Scan → Quiz pipeline (in progress) |
| **Next blocker** | Frontend/mobile auth wiring + real OCR/AI integration |

### Phase-wise progress

| Phase | Scope | Status |
|-------|-------|--------|
| **Foundation** | Monorepo, DB schema, feature flags, auth backend, admin | ✅ ~90% |
| **Phase 1** | Scan upload → OCR → AI quiz → attempts → billing | 🟡 ~40% |
| **Phase 2** | Reports, review, subscriptions UI | 🔴 ~10% |
| **Phase 3** | Career goals module (behind flag) | 🔴 ~5% |

---

## 2. Tech Stack

### Backend (`backend/`)

| Layer | Technology | Version / Notes |
|-------|------------|-----------------|
| Language | Go | 1.24 |
| HTTP router | chi/v5 | v5.1.0 |
| Database | PostgreSQL | via pgx/v5 |
| Cache | Valkey / Redis | go-redis/v9 (Redis protocol) |
| Auth | JWT + bcrypt | golang-jwt/jwt/v5, golang.org/x/crypto |
| IDs | UUID | google/uuid |

### Frontend (`frontend/`)

| Layer | Technology | Version |
|-------|------------|---------|
| Framework | Next.js (App Router) | ^16.2.7 |
| UI | React | ^19.0.0 |
| Language | TypeScript | ^5.6.0 |

### Mobile (`mobile/`)

| Layer | Technology | Version |
|-------|------------|---------|
| Framework | Expo + Expo Router | ^57.0.0 |
| Runtime | React Native | 0.86.0 |
| Language | TypeScript | ~6.0.3 |
| Local storage | AsyncStorage | upload queue (offline retry) |

### Shared contract (`shared/`)

| Item | Purpose |
|------|---------|
| `types/featureFlags.ts` | Flag keys shared between frontend + mobile |
| Backend `GET /me/features` | Single API contract for module toggles |

### Planned / not yet integrated

| Service | Purpose | Status |
|---------|---------|--------|
| S3 / MinIO | Temp image storage (24h lifecycle) | ❌ Not wired |
| Tesseract / Vision API | OCR text extraction | ❌ Not wired |
| OpenAI / Gemini | Real question generation | ❌ Stub generator only |
| Razorpay / PayU | Payments (backend hooks exist) | 🟡 Partial |
| Firebase FCM | Push notifications | 🟡 Structure only |
| Resend / AWS SES | Email notifications | 🟡 Structure only |

### Local dev dependencies

- PostgreSQL (port 5432)
- Valkey or Memurai / Redis (port 6379)
- Node.js (frontend + mobile)
- Go 1.24+ (backend)

---

## 3. Folder Structure

```
study-app/
├── package.json                 # npm workspaces root (frontend + mobile)
├── README.md                    # Setup guide + architecture overview
├── PROJECT_STATUS.md            # ← Ye file (final status reference)
├── SPRINT_STATUS_ANALYSIS.md    # Older sprint analysis (partially outdated)
│
├── backend/                     # Go API — source of truth
│   ├── cmd/api/main.go          # Entry point, route wiring
│   ├── go.mod
│   ├── test_integration.sh      # Auth flow integration tests
│   ├── migrations/
│   │   ├── 0001_feature_flags.sql
│   │   ├── 0002_ai_generation_logs.sql
│   │   ├── 0003_reconciled_schema.sql   # 40+ tables (full domain)
│   │   ├── 0004_notifications.sql
│   │   └── 0005_auth.sql
│   └── internal/
│       ├── auth/                # ✅ Register, login, JWT, password reset
│       ├── billing/             # 🟡 Checkout + Razorpay/PayU webhooks
│       ├── common/
│       │   ├── config/          # Env-based config
│       │   └── middleware/      # CORS, JWT auth, admin guard
│       ├── featureflags/        # ✅ Feature flag system
│       ├── notifications/       # ✅ Full module (+ README)
│       ├── quiz/                # 🟡 Models + prompt only (no HTTP routes)
│       └── scan/                # 🟡 Job pipeline + stub quiz generation
│
├── frontend/                    # Next.js web app
│   ├── app/
│   │   ├── layout.tsx           # FeatureFlagsProvider wrapper
│   │   ├── page.tsx             # Dashboard + nav
│   │   ├── scan/page.tsx        # Scan form (no auth header yet)
│   │   ├── reports/page.tsx     # Placeholder
│   │   └── admin/features/      # ✅ Admin flag dashboard
│   ├── components/
│   │   └── FeatureGate.tsx      # Conditional UI by flag
│   ├── lib/featureFlags.tsx     # Context + hook
│   └── types/featureFlags.ts    # Flag types (mirrors shared/)
│
├── mobile/                      # Expo React Native app
│   ├── app/
│   │   ├── _layout.tsx          # Tab navigator + flag-gated Goals tab
│   │   └── scan.tsx             # Scan + offline upload queue
│   ├── hooks/useFeatureFlags.ts
│   ├── lib/
│   │   ├── config.ts            # API_URL
│   │   └── uploadQueue.ts       # AsyncStorage retry queue
│   └── app.json
│
├── shared/
│   └── types/featureFlags.ts    # Shared flag contract
│
└── docs/                        # (optional) architecture notes
```

---

## 4. Module Status — Kya Ho Gaya, Kya Baki Hai

### ✅ Complete / Working

| Module | Backend | Frontend | Mobile | Notes |
|--------|---------|----------|--------|-------|
| **Feature flags** | ✅ | ✅ | ✅ | Rollout %, admin toggle, `<FeatureGate>` |
| **Database schema** | ✅ | — | — | 40+ tables across 8 domains |
| **Auth (backend)** | ✅ | — | — | Register, login, refresh, reset password, JWT |
| **JWT middleware** | ✅ | — | — | RequireAuth + RequireAdmin |
| **Admin dashboard** | ✅ | ✅ | — | `/admin/features` — flags, AI cost columns |
| **Health check** | ✅ | — | — | `GET /health` (DB + cache ping) |
| **Notifications (backend)** | ✅ | — | — | Queue, worker, preferences, rate limits |
| **Config management** | ✅ | — | — | Centralized env loading |

### 🟡 Partial — Structure hai, production flow incomplete

| Module | Done | Remaining |
|--------|------|-----------|
| **Scan pipeline** | Job create, upload endpoint, consent, content-hash cache, stub quiz gen | S3/MinIO upload, OCR worker, real image processing, background job queue |
| **Quiz generation** | Models, prompt builder, integrated in scan service via `defaultGenerator` | Dedicated `/quizzes` API, real OpenAI/Gemini, multi-question sets |
| **Content caching** | Redis + PostgreSQL `content_cache`, hash lookup | Cache invalidation, embedding similarity QA |
| **Billing** | Checkout response, Razorpay/PayU webhook handlers | Frontend checkout UI, subscription lifecycle, plan management |
| **Notifications (delivery)** | FCM/email service structure, circuit breaker | Firebase + Resend/SES API keys and live sending |
| **Frontend scan** | Basic form UI | Auth token in requests, file upload, status polling |
| **Mobile scan** | Upload queue + retry | Auth wiring, camera/gallery picker, real multipart upload |

### ❌ Not Started / Schema Only

| Module | DB Schema | Backend API | Frontend UI | Mobile UI |
|--------|-----------|-------------|-------------|-----------|
| **Quiz attempts** | ✅ | ❌ | Placeholder `/reports` | Placeholder tab |
| **Quiz reports / review** | ✅ | ❌ | ❌ | ❌ |
| **Career goals** | ✅ | ❌ | Links only (flag-gated) | Tab hidden by default |
| **Daily practice** | ✅ | ❌ | ❌ | ❌ |
| **Books / chapters CMS** | ✅ | ❌ | ❌ | ❌ |
| **Frontend auth UI** | — | — | ❌ | ❌ |
| **Email verification** | Column exists | 🟡 Token flow partial | ❌ | ❌ |
| **OpenAPI / Swagger docs** | — | ❌ | — | — |

---

## 5. API Endpoints (Current)

### Public

| Method | Path | Status |
|--------|------|--------|
| GET | `/health` | ✅ |
| POST | `/auth/register` | ✅ |
| POST | `/auth/login` | ✅ |
| POST | `/auth/refresh` | ✅ |
| POST | `/auth/forgot-password` | ✅ |
| POST | `/auth/reset-password` | ✅ |

### Authenticated (Bearer JWT required)

| Method | Path | Status |
|--------|------|--------|
| GET | `/auth/me` | ✅ |
| POST | `/auth/change-password` | ✅ |
| GET | `/me/features` | ✅ |
| POST | `/scan/jobs` | ✅ |
| POST | `/scan/upload` | ✅ (multipart; OCR not live) |
| GET | `/scan/jobs/{jobID}` | ✅ |
| GET | `/scan/uploads/{jobID}/status` | ✅ |
| POST | `/billing/checkout` | 🟡 |
| POST | `/billing/webhook/razorpay` | 🟡 |
| POST | `/billing/webhook/payu` | 🟡 |
| POST | `/auth/devices/register` | ✅ |
| GET | `/user/preferences` | ✅ |
| PUT | `/user/preferences` | ✅ |
| POST | `/notifications/send` | ✅ |
| POST | `/webhooks/email/resend` | 🟡 |

### Admin only

| Method | Path | Status |
|--------|------|--------|
| GET | `/admin/features` | ✅ |
| POST | `/admin/features` | ✅ |

### Planned (not mounted in `main.go`)

| Module | Example routes |
|--------|----------------|
| Quiz | `GET /quizzes/{id}`, `POST /quizzes/{id}/attempts` |
| Career goals | `GET /goals`, `POST /goals/select` |
| Reports | `GET /reports`, `GET /attempts/{id}/review` |

---

## 6. Workflows

### 6.1 System Architecture

```mermaid
flowchart TB
    subgraph clients [Clients]
        WEB[Frontend - Next.js]
        MOB[Mobile - Expo]
    end

    subgraph backend [Go Backend :8080]
        AUTH[Auth Module]
        FF[Feature Flags]
        SCAN[Scan Module]
        QUIZ[Quiz - stub]
        BILL[Billing]
        NOTIF[Notifications]
    end

    subgraph data [Data Layer]
        PG[(PostgreSQL)]
        CACHE[(Valkey / Redis)]
    end

    subgraph external [External - Not Connected]
        S3[S3 / MinIO]
        OCR[OCR Engine]
        AI[OpenAI / Gemini]
        PAY[Razorpay / PayU]
        FCM[Firebase FCM]
    end

    WEB --> AUTH
    MOB --> AUTH
    WEB --> FF
    MOB --> FF
    WEB --> SCAN
    MOB --> SCAN

    AUTH --> PG
    FF --> PG
    SCAN --> PG
    SCAN --> CACHE
    SCAN --> QUIZ
    BILL --> PG
    NOTIF --> PG

    SCAN -.-> S3
    SCAN -.-> OCR
    QUIZ -.-> AI
    BILL -.-> PAY
    NOTIF -.-> FCM
```

### 6.2 Feature Flag Flow

Modules ko bina redeploy ke on/off kiya ja sakta hai.

```mermaid
sequenceDiagram
    participant Admin
    participant Backend
    participant DB as PostgreSQL
    participant Client as Frontend / Mobile

    Admin->>Backend: POST /admin/features {key, enabled, rollout%}
    Backend->>DB: UPDATE feature_flags
    Note over DB: career_goals_module = OFF by default

    Client->>Backend: GET /me/features (JWT)
    Backend->>DB: Resolve flags for user_id
    DB-->>Backend: scan_quiz=ON, career_goals=OFF
    Backend-->>Client: {scan_quiz_module: true, career_goals_module: false}

    alt Flag ON
        Client->>Client: Show Career Goals nav / tab
    else Flag OFF
        Client->>Client: Hide via FeatureGate / href:null
    end
```

**Configured flags:**

| Flag key | Default | Phase |
|----------|---------|-------|
| `scan_quiz_module` | ON (100% rollout) | Phase 1 — core |
| `career_goals_module` | OFF (0% rollout) | Phase 3 — future |

### 6.3 Scan → Quiz Flow (Current — Stub)

Abhi real OCR/AI nahi; placeholder text se quiz generate hota hai.

```mermaid
sequenceDiagram
    participant User
    participant Client as Web / Mobile
    participant API as Scan Service
    participant Cache as Valkey
    participant DB as PostgreSQL

    User->>Client: Accept ToS + submit scan job
    Client->>API: POST /scan/jobs (needs JWT)
    API->>API: Validate board (ncert / state)
    API->>DB: Record consent audit log
    API->>DB: Create scan_job + scan_page

    Note over API: Source text = placeholder (no OCR yet)
    API->>API: SHA256 content hash

    alt Cache hit
        API->>Cache: GET content_hash
        Cache-->>API: quiz_id
        API-->>Client: Job complete (from cache)
    else Cache miss
        API->>DB: Create ai_generation_log
        API->>API: defaultGenerator (stub, not real AI)
        API->>DB: Save question, options, explanation, quiz
        API->>DB: Write content_cache
        API->>Cache: SET content_hash → quiz_id (24h TTL)
        API-->>Client: Job complete + quiz_id
    end

    Note over API,DB: Raw image NOT stored long-term (design intent)
```

### 6.4 Auth Flow (Backend ready, clients not wired)

```mermaid
sequenceDiagram
    participant User
    participant Client
    participant API as Auth Service
    participant DB as PostgreSQL

    User->>Client: Register / Login
    Client->>API: POST /auth/register or /auth/login
    API->>DB: Create user / verify bcrypt hash
    API-->>Client: {access_token, refresh_token}

    Note over Client: ⚠️ Frontend/mobile abhi token store nahi karte

    Client->>API: GET /me/features
    Note right of Client: Authorization: Bearer {token}
    API->>API: JWT verify → user_id in context
    API-->>Client: Resolved flags

    Client->>API: POST /scan/jobs
    Note right of Client: ⚠️ Web scan page abhi bina token ke call karta hai → 401
```

### 6.5 Mobile Offline Upload Queue

```mermaid
flowchart LR
    A[User taps Queue Upload] --> B{Terms accepted?}
    B -->|No| C[Alert: Consent required]
    B -->|Yes| D[Enqueue in AsyncStorage]
    D --> E[POST /scan/upload]
    E -->|Success| F[Status: ready]
    E -->|Fail| G{attempts < 5?}
    G -->|Yes| H[Exponential backoff retry]
    H --> E
    G -->|No| I[Status: failed — manual Retry button]
```

### 6.6 Target Production Flow (Roadmap — not built yet)

```mermaid
flowchart LR
    A[Camera / Gallery] --> B[S3 temp upload]
    B --> C[OCR Worker]
    C --> D[Content hash]
    D --> E{Cache hit?}
    E -->|Yes| F[Serve cached quiz]
    E -->|No| G[AI generates questions]
    G --> H[Save to DB + cache]
    H --> I[User takes quiz]
    I --> J[Score + weak topics report]
    J --> K[Optional: Career goals practice]
```

---

## 7. Database Domains (Schema Ready)

Migration `0003_reconciled_schema.sql` covers:

| Domain | Tables | Backend wired |
|--------|--------|---------------|
| Identity | users, user_profiles | ✅ auth |
| Feature flags | feature_flags, user_feature_overrides | ✅ |
| Content | books, chapters | ❌ |
| Scan | scan_jobs, scan_pages | ✅ |
| Quiz | questions, options, explanations, quizzes | 🟡 via scan only |
| Attempts | quiz_attempts, answers, reports | ❌ |
| Career goals | career_goals, student_goals, daily_practice_sets, … | ❌ |
| Billing | plans, subscriptions, payments, payment_events | 🟡 |
| Admin / audit | job_events, admin_actions, audit_logs, content_flags | 🟡 partial |
| AI logs | ai_generation_logs | ✅ written by scan |
| Cache | content_cache | ✅ |
| Notifications | fcm_device_tokens, notification_jobs, … | ✅ |

---

## 8. Environment Variables & Config Validation

Startup par `config.Load()` + `config.Validate()` chalti hai (`cmd/api/main.go`).

### Validation behaviour

| Environment | Default `JWT_SECRET` | `localhost` DB | Missing `RAZORPAY_KEY_ID` | Result |
|-------------|----------------------|----------------|---------------------------|--------|
| `development` | Allowed | Allowed | Allowed | ✅ Starts + **warning logs** |
| `staging` | Blocked | Blocked* | Blocked | ❌ Fail fast (production rules) |
| `production` | Blocked | Blocked* | Blocked | ❌ Fail fast, all errors at once |

\*Override: `ALLOW_LOCALHOST_DB=true` (local prod-like testing only)

### Backend variables

| Variable | Required (prod) | Default (dev) | Notes |
|----------|-----------------|---------------|-------|
| `ENVIRONMENT` | Yes | `development` | `production` triggers strict validation |
| `PORT` | Yes | `8080` | HTTP listen port |
| `DATABASE_URL` | Yes | `postgres://localhost:5432/studyapp?sslmode=disable` | No localhost in prod |
| `VALKEY_ADDR` | Yes | `localhost:6379` | Valkey / Redis |
| `JWT_SECRET` | Yes | `dev-secret-change-in-production` | **Blocked in prod** |
| `RAZORPAY_KEY_ID` | Yes | — | Razorpay public key |
| `RAZORPAY_WEBHOOK_SECRET` | Recommended | — | Webhook signature verify |
| `ALLOW_LOCALHOST_DB` | — | — | `true` to allow localhost DB in prod |

### Client variables

| Variable | Used by | Default (dev) |
|----------|---------|---------------|
| `NEXT_PUBLIC_API_URL` | Frontend | `http://localhost:8080` |
| `EXPO_PUBLIC_API_URL` | Mobile | `http://localhost:8080` |

Template: `backend/.env.example` · Full guide: `backend/docs/PRODUCTION.md`

---

## 9. Run Commands

### Development (no env setup needed)

```bash
# Backend (Postgres + Valkey/Redis + migrations required)
cd backend
go mod tidy
go run cmd/api/main.go

# Frontend
cd frontend
npm install
npm run dev          # http://localhost:3000

# Mobile
cd mobile
npm install --legacy-peer-deps
npx expo start

# Integration tests (backend must be running)
cd backend
bash test_integration.sh
```

**Root workspace shortcuts** (from repo root):

```bash
npm run backend
npm run frontend
npm run mobile
```

### Production backend start

**Linux / macOS:**

```bash
cd backend

export ENVIRONMENT=production
export PORT=8080
export DATABASE_URL="postgres://studyapp:YOUR_PASSWORD@db.example.com:5432/studyapp?sslmode=require"
export VALKEY_ADDR="cache.example.com:6379"
export JWT_SECRET="$(openssl rand -base64 32)"
export RAZORPAY_KEY_ID="rzp_live_xxxxxxxx"
export RAZORPAY_WEBHOOK_SECRET="whsec_xxxxxxxx"   # recommended

go run cmd/api/main.go
# Verify: curl http://localhost:8080/health
```

**Windows PowerShell:**

```powershell
cd backend

$env:ENVIRONMENT = "production"
$env:PORT = "8080"
$env:DATABASE_URL = "postgres://studyapp:YOUR_PASSWORD@db.example.com:5432/studyapp?sslmode=require"
$env:VALKEY_ADDR = "cache.example.com:6379"
$env:JWT_SECRET = [Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }))
$env:RAZORPAY_KEY_ID = "rzp_live_xxxxxxxx"
$env:RAZORPAY_WEBHOOK_SECRET = "whsec_xxxxxxxx"

go run cmd/api/main.go
```

**Helper scripts** (edit placeholders inside first):

```bash
./backend/scripts/run-production.sh       # Linux / macOS
.\backend\scripts\run-production.ps1      # Windows
```

**Build binary:**

```bash
cd backend
go build -o studyapp-api ./cmd/api
./studyapp-api    # same env vars required
```

See `backend/docs/PRODUCTION.md` for pre-flight checklist, common errors, and client env vars.

---

## 10. Remaining Work — Priority Order

### P0 — Blockers (Phase 1 MVP)

1. **Frontend + mobile auth UI** — login/register, token storage, attach `Authorization` header to all API calls
2. **S3/MinIO integration** — temp image upload with 24h lifecycle
3. **OCR worker** — Tesseract local or cloud Vision API
4. **Real AI integration** — replace `defaultGenerator` with OpenAI/Gemini
5. **Quiz attempts API** — mount routes in `main.go`, scoring + reports

### P1 — Core product completion

6. Reports / review UI (frontend + mobile)
7. Billing checkout UI + subscription status
8. File upload on web scan page (multipart)
9. Camera/gallery picker on mobile
10. Notification worker process + FCM/SES live credentials

### P2 — Phase 3 (behind `career_goals_module` flag)

11. Career goals backend module
12. Daily practice sets
13. Skill gap analysis
14. Adaptive weak-topic engine

### P3 — Production hardening

15. Config validation on startup (fail if `JWT_SECRET` is default in prod)
16. OpenAPI documentation
17. Structured error responses
18. E2E test suite
19. CI/CD pipelines (Vercel + EAS + backend deploy)

---

## 11. Known Gaps & Risks

| Issue | Impact | Location |
|-------|--------|----------|
| Frontend scan calls API without JWT | 401 errors in normal use | `frontend/app/scan/page.tsx` |
| Mobile `authToken = null` | Feature flags always use defaults | `mobile/app/_layout.tsx` |
| No real OCR/AI | Quiz quality is placeholder only | `backend/internal/scan/service.go` |
| Quiz HTTP module not mounted | No direct quiz/attempt endpoints | `backend/cmd/api/main.go` |
| `SPRINT_STATUS_ANALYSIS.md` outdated | Says auth is missing — it is now done | Root (ignore auth gaps there) |
| Notifications migration SQL syntax | May need Postgres-compatible indexes | `0004_notifications.sql` |

---

## 12. Completion Scorecard

```
Foundation          ████████████████████░░  90%
Auth (backend)      ██████████████████████ 100%
Feature flags       ██████████████████████ 100%
Admin dashboard     ████████████████████░░  85%
Scan pipeline       ████████░░░░░░░░░░░░░░  35%
AI / Quiz gen       ████░░░░░░░░░░░░░░░░░░  20%
Quiz attempts       ██░░░░░░░░░░░░░░░░░░░░  10%
Billing             ██████░░░░░░░░░░░░░░░░  30%
Notifications       ████████████████░░░░░░  75%
Frontend UX         ████████░░░░░░░░░░░░░░  35%
Mobile UX           ██████████░░░░░░░░░░░░  40%
Career goals        █░░░░░░░░░░░░░░░░░░░░░   5%
─────────────────────────────────────────────
Overall MVP         ███████████░░░░░░░░░░░  ~55%
```

---

## 13. Document History

| Date | Change |
|------|--------|
| 2026-07-05 | Initial final status document created |

---

*StudyApp · Production Roadmap v1.0 · Monorepo architecture*
