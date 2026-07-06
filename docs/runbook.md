# StudyApp Runbook

Operational guide for on-call engineers responding to production incidents.

## Quick links

| Resource | Location |
|----------|----------|
| Health (detailed) | `GET /health` |
| Readiness probe | `GET /ready` |
| Metrics | `GET /metrics` (internal network only) |
| Admin panel | `/admin/*` (admin JWT required) |
| Grafana | Configure Prometheus scrape → dashboards |

---

## Incident Response

### 1. High Scan Failure Rate

**Symptoms:** `scan_jobs_total{status="failed"}` exceeds 20% of completed jobs; users report stuck or failed scans.

**Steps:**

1. Check OCR worker logs: `docker logs studyapp-backend 2>&1 | grep -i ocr`
2. Verify OCR provider status:
   - **Tesseract:** local process — check disk space and binary availability
   - **Google Vision:** GCP Console → APIs & Services → Cloud Vision API quotas/errors
3. Check storage connectivity: `curl -s http://localhost:8080/health | jq .checks.storage`
4. Check S3/MinIO bucket permissions and lifecycle rules if using object storage
5. **Fallback:** set `OCR_PROVIDER=stub` in environment and restart backend (generates placeholder quizzes — use only as emergency)
6. Re-queue failed jobs via admin: `POST /admin/jobs/{id}/retry` for each failed job ID from `/admin/jobs?status=failed`

**Resolution criteria:** Scan failure rate below 5% for 15 minutes.

---

### 2. Database Connection Exhausted

**Symptoms:** `/health` returns `"status": "down"` with `checks.database.status: "down"`; API returns 503; logs show `too many connections` or connection timeouts.

**Steps:**

1. Confirm Postgres is reachable: `pg_isready -d "$DATABASE_URL"`
2. Check active connections: `SELECT count(*) FROM pg_stat_activity;`
3. Identify long-running queries: `SELECT pid, now()-query_start AS duration, query FROM pg_stat_activity WHERE state != 'idle' ORDER BY duration DESC LIMIT 10;`
4. Terminate stuck queries if safe: `SELECT pg_terminate_backend(pid);`
5. Scale connection pool: reduce `pgxpool` max connections in app config or increase Postgres `max_connections`
6. Restart backend pods/containers one at a time to release leaked connections
7. If disk full on DB host, expand volume and run `VACUUM` on large tables

**Resolution criteria:** `/ready` returns 200; database latency under 50ms in `/health`.

---

### 3. Cache (Valkey/Redis) Unavailable

**Symptoms:** `/health` shows `checks.cache.status: "down"`; rate limiting and feature flags may fail; scan cache misses spike.

**Steps:**

1. Ping cache: `redis-cli -u "$VALKEY_ADDR" ping`
2. Check memory usage: `redis-cli INFO memory`
3. If OOM, increase instance memory or set eviction policy `allkeys-lru`
4. Restart Valkey/Memurai service
5. Backend can operate without cache (degraded) — scan cache falls back to Postgres; rate limits may be bypassed if Redis is down

**Resolution criteria:** Cache check returns `"status": "ok"`.

---

### 4. Payment Webhook Failures

**Symptoms:** Users paid but subscription not active; `payment_events_total{status="failed"}` increasing; Razorpay dashboard shows webhook delivery failures.

**Steps:**

1. Verify webhook secrets: `RAZORPAY_WEBHOOK_SECRET` and `PAYU_SALT` match provider dashboards
2. Check backend logs for `invalid webhook signature` or `webhook processing failed`
3. Confirm webhook URL is publicly reachable over HTTPS: `https://api.studyapp.in/billing/webhook/razorpay`
4. In Razorpay Dashboard → Webhooks → select failed event → **Resend**
5. For PayU, replay callback from merchant dashboard or contact PayU support
6. Manually verify payment in DB: `SELECT * FROM payments WHERE provider_order_id = '...'`
7. If payment marked completed but no subscription, run admin SQL or contact engineering to call `UpsertActiveSubscription`

**Resolution criteria:** Webhook events processed; affected users show active subscription in `/users/me/subscription`.

---

### 5. High AI Cost Spike

**Symptoms:** `ai_tokens_used_total` or admin AI cost dashboard shows sudden spike; `ai_generation_logs.cost_estimate` elevated.

**Steps:**

1. Open admin AI costs: `GET /admin/ai-costs?from=YYYY-MM-DD&to=YYYY-MM-DD`
2. Check for abuse: unusual user with many scan jobs in `/admin/users` and `/admin/jobs`
3. Review cache hit rate — low hits mean more AI calls: `scan_cache_hits_total` vs `scan_cache_misses_total`
4. Temporarily reduce scan limits via billing plans or suspend abusive accounts
5. Switch AI provider to stub in emergency: set `AI_PROVIDER=stub` and restart
6. Enable stricter per-user AI rate limits if configured

**Resolution criteria:** Daily AI cost returns to baseline; no single user exceeding 10× normal scan volume.

---

### 6. Notification Delivery Backlog

**Symptoms:** `notification_queue_depth` gauge high; users not receiving push/email; worker logs show repeated failures.

**Steps:**

1. Check queue depth metric and Redis keys: `LLEN notif:queue`, `LLEN notif:retry`, `LLEN notif:dlq`
2. Review worker logs: `docker logs studyapp-backend 2>&1 | grep notification`
3. For FCM failures: verify Firebase credentials path and token validity
4. For email failures: check SES/Resend API status and bounce rates
5. Inspect dead-letter queue jobs in Redis (`notif:dlq`) for permanent failures
6. Restart notification worker by restarting backend container

**Resolution criteria:** Queue depth near zero; test push via authenticated `POST /notifications/send`.

---

### 7. API Latency Degradation

**Symptoms:** `http_request_duration_seconds` p95 > 2s; load balancer timeouts; user complaints of slow app.

**Steps:**

1. Check `/health` for degraded storage or high DB latency
2. Review Grafana dashboard for slow endpoints (path label in histogram)
3. Check Postgres slow queries and missing indexes
4. Verify Valkey latency in health check
5. Scale backend replicas horizontally if CPU-bound
6. Enable connection pooling review — ensure pool size matches load

**Resolution criteria:** p95 latency under 500ms for core endpoints (`/health`, `/me/features`, `/scan/jobs`).

---

### 8. Authentication Outage

**Symptoms:** All login/register requests fail with 401/500; JWT validation errors in logs.

**Steps:**

1. Verify `JWT_SECRET` has not changed across all backend instances (token invalidation if rotated incorrectly)
2. Check database connectivity — auth requires Postgres for user lookup
3. Review rate limit on `/auth/*` — legitimate traffic may be blocked if IP shared (NAT)
4. Check audit logs for brute-force patterns
5. If secret compromised, rotate `JWT_SECRET`, force password resets, invalidate sessions

**Resolution criteria:** Login and token refresh succeed; audit logs recording login events.

---

### 9. Storage Upload Failures

**Symptoms:** Scan uploads return 400/500; `checks.storage` degraded in `/health`; OCR cannot fetch page images.

**Steps:**

1. Check `/health` storage check latency and status
2. Verify MinIO/S3 bucket exists and credentials are valid
3. Check disk space on local storage (`STORAGE_LOCAL_DIR`) in dev/single-node setups
4. Review upload validation errors — only JPEG/PNG/WebP accepted (magic-byte check)
5. Confirm `MAX_UPLOAD_SIZE_MB` and nginx/proxy body size limits align (≥ 11MB)

**Resolution criteria:** Test upload via `POST /scan/upload` succeeds; storage check `"status": "ok"`.

---

## Escalation

| Severity | Response time | Escalate to |
|----------|---------------|-------------|
| P1 — Full outage (DB down, no auth) | 15 min | Engineering lead + infra |
| P2 — Feature degraded (scans, payments) | 1 hour | On-call engineer |
| P3 — Non-critical (notifications delayed) | Next business day | Product + engineering |

## Post-incident

1. Document timeline in incident channel
2. Update this runbook if steps were missing or incorrect
3. Add monitoring alert if detection was delayed
