PRAGMA foreign_keys=off;

DROP TABLE IF EXISTS deployment_rollback_payloads;
DROP INDEX IF EXISTS idx_rollback_events_deployment;
DROP INDEX IF EXISTS idx_deployment_jobs_app_running;
DROP INDEX IF EXISTS idx_deployment_jobs_app_status;
DROP INDEX IF EXISTS idx_deployment_jobs_status_next_attempt;
DROP TABLE IF EXISTS deployment_jobs;

CREATE TABLE IF NOT EXISTS deployments_phase8_down (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  app_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  status TEXT NOT NULL,
  trigger_type TEXT NOT NULL,
  commit_sha TEXT,
  commit_message TEXT,
  commit_author TEXT,
  branch TEXT,
  site_url TEXT,
  failure_reason TEXT,
  correlation_id TEXT,
  release_version_id INTEGER,
  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
  updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
  started_at INTEGER,
  finished_at INTEGER
);

INSERT INTO deployments_phase8_down (
  id, app_id, user_id, status, trigger_type, commit_sha, commit_message, commit_author, branch, site_url, failure_reason, correlation_id,
  release_version_id, created_at, updated_at, started_at, finished_at
)
SELECT
  id, app_id, user_id, status, trigger_type, commit_sha, commit_message, commit_author, branch, site_url, failure_reason, correlation_id,
  release_version_id, created_at, updated_at, started_at, finished_at
FROM deployments;

DROP TABLE deployments;
ALTER TABLE deployments_phase8_down RENAME TO deployments;

CREATE INDEX IF NOT EXISTS idx_deployments_app_created
  ON deployments(app_id, created_at DESC);

PRAGMA foreign_keys=on;
