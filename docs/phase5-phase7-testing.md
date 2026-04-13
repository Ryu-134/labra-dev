# Phase 5 + 7 Testing Guide (Rollback/Versioning + Observability)

This guide explains how testing is set up for this phase, how to run it locally, and what CI runs automatically.

## 1) What we test

### Phase 5 (Rollback + Versioning)

- Successful deployments create release snapshots.
- App `current_release` pointer updates to newest release.
- Retention policy marks older releases as not retained.
- Rollback endpoint creates a rollback deployment.
- Rollback pointer switch updates app current release.
- Rollback events are persisted.

### Phase 7 (Observability)

- Observability summary returns:
  - status/trigger distributions
  - release count
  - recent durations
  - recent failures
  - rollback history
  - richer health metrics
- Log query endpoint supports local query mode.
- CloudWatch mode is explicitly reported as available/unavailable.

## 2) Automated test files

Backend tests are located in `labra-backend/internal/api/handlers/`:

- `phase4_webhook_test.go` (existing regression coverage)
- `phase6_env_health_test.go` (existing env/health coverage)
- `phase5_phase7_rollback_observability_test.go` (new for this phase)

Tests use in-memory SQLite so they run quickly and do not require external services.

## 3) Run all tests locally

From repo root:

```bash
cd labra-backend
mkdir -p .gocache
GOCACHE=$(pwd)/.gocache go test ./...
rm -rf .gocache
```

Frontend type checks:

```bash
cd ../labra-frontend
npm run check
```

## 4) Manual API smoke checks

Start backend:

```bash
cd labra-backend
make run
```

Assume `app_id=1`, user header `X-User-ID: 1`.

### List releases

```bash
curl -s http://localhost:8080/v1/apps/1/releases -H 'X-User-ID: 1'
```

### Trigger rollback to a specific release

```bash
curl -s -X POST http://localhost:8080/v1/apps/1/rollback \
  -H 'Content-Type: application/json' \
  -H 'X-User-ID: 1' \
  -d '{"target_release_id": 1, "reason":"manual verification"}'
```

### Rollback history

```bash
curl -s http://localhost:8080/v1/apps/1/rollbacks -H 'X-User-ID: 1'
```

### Observability summary

```bash
curl -s http://localhost:8080/v1/apps/1/observability -H 'X-User-ID: 1'
```

### Log query (local mode)

```bash
curl -s "http://localhost:8080/v1/apps/1/observability/log-query?q=build&source=local" -H 'X-User-ID: 1'
```

### Log query (cloudwatch-compatible toggle)

Disabled mode response:

```bash
curl -s "http://localhost:8080/v1/apps/1/observability/log-query?q=build&source=cloudwatch" -H 'X-User-ID: 1'
```

Enabled mode (uses local compatibility behavior in this project):

```bash
export CLOUDWATCH_QUERY_ENABLED=true
```

## 5) CI/CD automation

A minimal GitHub Actions pipeline is included at:

- `.github/workflows/ci.yml`

It runs on PRs and pushes:

1. Backend Tests job:
- `go test ./...`

2. Frontend Type Check job:
- `npm ci`
- `npm run check`

This ensures core backend and frontend quality checks run automatically before merge.

## 6) How this testing design helps you

- Unit/integration blend: handler tests validate behavior close to real API logic.
- Fast feedback: in-memory DB tests run quickly.
- Regression safety: older phase tests still run with new phase tests.
- CI gate: no manual remembering required; checks run automatically on every PR.
