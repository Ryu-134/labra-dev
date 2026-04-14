ALTER TABLE deployments
  ADD COLUMN failure_category TEXT;

ALTER TABLE deployments
  ADD COLUMN retryable INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS deployment_jobs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  deployment_id INTEGER NOT NULL,
  app_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'queued',
  attempt_count INTEGER NOT NULL DEFAULT 0,
  max_attempts INTEGER NOT NULL DEFAULT 3,
  next_attempt_at INTEGER NOT NULL DEFAULT (unixepoch()),
  last_error TEXT,
  error_category TEXT,
  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
  updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
  started_at INTEGER,
  finished_at INTEGER,
  claimed_by TEXT,
  UNIQUE(deployment_id)
);

CREATE INDEX IF NOT EXISTS idx_deployment_jobs_status_next_attempt
  ON deployment_jobs(status, next_attempt_at ASC);

CREATE INDEX IF NOT EXISTS idx_deployment_jobs_app_status
  ON deployment_jobs(app_id, status);

CREATE UNIQUE INDEX IF NOT EXISTS idx_deployment_jobs_app_running
  ON deployment_jobs(app_id)
  WHERE status = 'running';

CREATE TABLE IF NOT EXISTS deployment_rollback_payloads (
  deployment_id INTEGER PRIMARY KEY,
  from_release_id INTEGER NOT NULL,
  target_release_id INTEGER NOT NULL,
  reason TEXT,
  created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_rollback_events_deployment
  ON rollback_events(deployment_id);
