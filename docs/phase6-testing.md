# Phase 6 Testing Guide (Env Vars + Health)

This document explains what we added for Phase 6 and how to test it.

## 1) Automated backend tests

Run from `labra-backend/`:

```bash
mkdir -p .gocache
GOCACHE=$(pwd)/.gocache go test ./...
rm -rf .gocache
```

### New test coverage

- `internal/api/handlers/phase6_env_health_test.go`
- Env var CRUD behavior.
- Secret masking in API responses.
- Env var injection in deployment flow.
- Success/failure metric persistence.
- Health summary endpoint shape and values.

### Existing Phase 4 coverage still included

- `internal/api/handlers/phase4_webhook_test.go`
- Signature validation, dedupe behavior, webhook-triggered deployments, and history metadata.

## 2) Manual API verification (optional but recommended)

Start backend first:

```bash
cd labra-backend
make run
```

Assume `X-User-ID: 1` and app ID `1`.

### Create env var

```bash
curl -s -X POST http://localhost:8080/v1/apps/1/env-vars \
  -H 'Content-Type: application/json' \
  -H 'X-User-ID: 1' \
  -d '{"key":"API_TOKEN","value":"secret-value","is_secret":true}'
```

Expected: `value` returned as `********`.

### List env vars

```bash
curl -s http://localhost:8080/v1/apps/1/env-vars -H 'X-User-ID: 1'
```

Expected: secret values stay masked; non-secret values are visible.

### Health summary

```bash
curl -s http://localhost:8080/v1/apps/1/health -H 'X-User-ID: 1'
```

Expected fields include:
- `current_url`
- `latest_deploy_status`
- `last_successful_deploy`
- `metrics` (success/failure/total/success_rate)
- `alarm_state` (when `LABRA_ALARM_STATE` env var exists)
- `health_indicator`

## 3) Frontend verification

Open app details page: `/apps/:id`

Check these tabs:
- `Deployments`: existing history still works.
- `Env Vars`: create/update/delete variables and verify masking.
- `Health`: verify status, URL, last success, alarm state, and indicator.

## 4) Test philosophy used

- Handler tests use in-memory SQLite with schema setup for realistic integration testing.
- Tests validate both success paths and important edge behavior (duplicates, masking, metrics).
- We preserved existing Phase 4 tests so regressions are caught while expanding features.
