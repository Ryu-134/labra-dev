# Phase 8 Testing Guide (Reliability Queue + Retries)

This guide explains how to test Phase 8 end-to-end and what each test is proving.

## 1) What Phase 8 added

- Deployment job queue (`deployment_jobs`).
- Worker-driven deploy execution (manual, webhook, rollback all enqueue jobs).
- Retry + backoff for transient failures.
- Concurrency control (single running deployment per app at a time).
- Failure categorization (`failure_category`) and retryability (`retryable`) on deployments.
- Queue status API endpoint: `GET /v1/deploys/:id/queue`.

## 2) Automated backend tests

Backend tests run from `labra-backend/` and use in-memory SQLite.

```bash
cd labra-backend
mkdir -p .gocache
GOCACHE=$(pwd)/.gocache go test ./...
rm -rf .gocache
```

### New Phase 8 test file

- `internal/api/handlers/phase8_queue_retries_test.go`

### What those tests cover

1. Deploy job is enqueued and processed by worker.
2. Transient failure retries with backoff and eventually succeeds.
3. Non-retryable configuration failures stop after one attempt.
4. Rollback path uses queue + rollback payload table + worker execution.
5. Queue enqueue is idempotent per deployment.
6. Queue status endpoint returns job state.

## 3) Frontend checks

Run frontend static/type checks:

```bash
cd labra-frontend
npm run check
```

If `npm` is not available in your shell, install Node.js first or use your existing Node manager profile in a login shell.

## 4) Manual smoke checks

Start backend:

```bash
cd labra-backend
make run
```

Assume `X-User-ID: 1` and app ID `1`.

### Trigger a deployment

```bash
curl -s -X POST http://localhost:8080/v1/apps/1/deploy -H 'X-User-ID: 1'
```

### Check deployment details

```bash
curl -s http://localhost:8080/v1/deploys/1 -H 'X-User-ID: 1'
```

Look for:
- `status`
- `failure_category`
- `retryable`

### Check queue status

```bash
curl -s http://localhost:8080/v1/deploys/1/queue -H 'X-User-ID: 1'
```

Look for:
- `job.status`
- `job.attempt_count`
- `job.max_attempts`
- `next_retry_in_seconds`

### Trigger rollback

```bash
curl -s -X POST http://localhost:8080/v1/apps/1/rollback \
  -H 'Content-Type: application/json' \
  -H 'X-User-ID: 1' \
  -d '{"reason":"manual phase8 verification"}'
```

## 5) Testing retries on purpose

Set an app env var:

- key: `LABRA_FORCE_TRANSIENT_FAILURES`
- value: `1`

Then trigger deploy. The first attempt fails, the next attempt should succeed.

For non-retryable simulation:

- key: `LABRA_FORCE_PERMANENT_FAILURE`
- value: `true`

Then trigger deploy. It should fail without retry loop.

## 6) CI/CD automation

CI workflow:

- `.github/workflows/ci.yml`

It runs automatically on pull requests and main/phase branches, and includes:

1. Backend tests: `go test ./...`
2. Frontend checks: `npm ci` + `npm run check`

This gives automatic regression protection for queue/retry behavior and UI integration.
