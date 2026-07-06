# Production Launch Checklist

Use this checklist before directing production traffic to StudyApp.

## Infrastructure

- [ ] PostgreSQL: production instance provisioned + automated backups enabled
- [ ] Valkey/Redis: production instance provisioned with persistence if required
- [ ] S3/MinIO: production bucket with lifecycle rules (temp object expiry)
- [ ] Domain + SSL certificate configured and valid
- [ ] Load balancer configured with health check on `GET /ready`
- [ ] Backend replicas behind load balancer (minimum 2 for availability)

## Security

- [ ] `JWT_SECRET`: not default, 32+ random characters
- [ ] All API keys: production values (not test/sandbox keys)
- [ ] `ALLOWED_ORIGINS`: only production domains (no `localhost` in prod)
- [ ] `METRICS_ALLOWED_IPS`: restricted to Prometheus scraper / private network
- [ ] HTTPS enforced (HTTP → HTTPS redirect at load balancer)
- [ ] Rate limiting tested under load (`bash load_test.sh`)
- [ ] Security headers verified (https://securityheaders.com or curl `-I`)

## Features

- [ ] `scan_quiz_module` flag: ON (100% rollout)
- [ ] `career_goals_module` flag: OFF (0% rollout until soft launch)
- [ ] Subscription plans seeded in production database
- [ ] Razorpay: production keys configured (not test mode)
- [ ] PayU credentials configured if using PayU

## Observability

- [ ] Prometheus scraping `GET /metrics` from internal network
- [ ] Grafana dashboards configured (HTTP, scan, AI, notifications, billing)
- [ ] Alert rules configured:
  - [ ] Error rate > 5% (5xx responses)
  - [ ] Latency p95 > 2s
  - [ ] Database down (`/ready` failing)
  - [ ] Scan failure rate > 20%
- [ ] Log aggregation working (JSON logs from backend)
- [ ] Runbook distributed to on-call team (`docs/runbook.md`)

## Compliance

- [ ] Terms of Service page live and linked from registration
- [ ] Privacy Policy page live
- [ ] Cookie consent implemented (if applicable for your markets)
- [ ] Data retention policy documented
- [ ] Admin access limited to necessary personnel
- [ ] Audit logs verified: test login, scan, and payment events recorded

## Testing

- [ ] End-to-end: register → login → scan → quiz → view reports
- [ ] Billing: test payment in Razorpay test mode against staging; production smoke test with small amount
- [ ] Webhook: verify Razorpay webhook delivery to production URL
- [ ] Notification: test push on real device; test email delivery
- [ ] Mobile: TestFlight (iOS) / internal APK (Android) tested against production API
- [ ] Load test: 100 concurrent users, p95 < 500ms on `/health` and authenticated endpoints
- [ ] Migration `0015_audit_log_metadata.sql` applied

## Backup & Recovery

- [ ] Database backup: automated daily with 30-day retention
- [ ] Recovery test: restore from backup verified in staging
- [ ] Incident playbooks reviewed by team (`docs/runbook.md`)
- [ ] Rollback procedure documented (previous container image + DB migration reversal plan)

---

**Sign-off**

| Role | Name | Date |
|------|------|------|
| Engineering | _________________ | _________ |
| Product | _________________ | _________ |
| Operations | _________________ | _________ |
