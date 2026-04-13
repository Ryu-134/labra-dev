# Phase 4 MVP Runbook

This runbook covers the Phase 4 auto-deploy MVP behavior:

- GitHub push webhook ingestion
- Signature verification
- Duplicate delivery handling
- Repo + branch routing
- Auto-triggered deployments
- Deployment history visibility

## Backend Environment

Set these values in `labra-backend/.env`:

```env
GH_CLIENT_ID=...
GH_CLIENT_SECRET=...
DB_URL=./labra.db
GITHUB_WEBHOOK_SECRET=replace-with-long-random-secret
```

## Start Backend

From `labra-backend/`:

```bash
make run
```

Expected endpoint base: `http://localhost:8080`.

## Configure GitHub Webhook

In your app repo:

1. Open `Settings -> Webhooks -> Add webhook`
2. Payload URL: `http://<your-public-url>/v1/webhooks/github`
3. Content type: `application/json`
4. Secret: exactly `GITHUB_WEBHOOK_SECRET`
5. Events: choose `Just the push event`

## Local Replay (No GitHub Required)

Use this to replay a signed push payload against local backend.

```bash
SECRET='replace-with-long-random-secret'
PAYLOAD='{"ref":"refs/heads/main","after":"abc123def456","repository":{"full_name":"owner/repo"},"head_commit":{"id":"abc123def456","message":"feat: update","author":{"name":"Casey"}}}'
SIG=$(printf '%s' "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" | sed 's/^.* //')

curl -i http://localhost:8080/v1/webhooks/github \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -H "X-GitHub-Delivery: local-replay-1" \
  -H "X-Hub-Signature-256: sha256=$SIG" \
  --data "$PAYLOAD"
```

## Expected Phase 4 Behavior

For a valid push payload:

- Request is accepted when signature is valid.
- Only apps matching `repository.full_name` are considered.
- Only apps matching pushed branch are eligible.
- Duplicate `X-GitHub-Delivery` values are ignored per app.
- Eligible apps enqueue webhook-triggered deployments.
- Commit metadata is attached to created deployments.

## Verify History + Logs

Use these endpoints with `X-User-ID`:

- `GET /v1/apps/:id/deploys` -> deployment history for app
- `GET /v1/deploys/:id` -> deployment details/status
- `GET /v1/deploys/:id/logs` -> deployment logs

These should expose:

- `trigger_type` (`manual` or `webhook`)
- `commit_sha`, `commit_message`, `commit_author`
- latest status and site URL

## Common Failure Cases

- Missing/invalid `X-Hub-Signature-256` -> request rejected.
- Missing `X-GitHub-Delivery` -> request rejected.
- Non-push GitHub event -> request accepted but ignored.
- Push to non-configured branch -> request accepted but ignored.

## MVP Scope Notes

Current Phase 4 is static app MVP only.

- Build type is `static`
- Auto deploy path reuses existing deployment execution flow
- Single environment (`dev`) focus
