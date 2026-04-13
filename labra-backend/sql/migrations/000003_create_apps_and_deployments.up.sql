CREATE TABLE IF NOT EXISTS apps (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  repo_full_name TEXT NOT NULL,
  branch TEXT NOT NULL DEFAULT 'main',
  build_type TEXT NOT NULL DEFAULT 'static',
  output_dir TEXT NOT NULL DEFAULT 'dist',
  root_dir TEXT,
  site_url TEXT,
  auto_deploy_enabled INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
  updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_apps_user_repo_branch
  ON apps(user_id, repo_full_name, branch);

CREATE INDEX IF NOT EXISTS idx_apps_repo
  ON apps(repo_full_name);

CREATE TABLE IF NOT EXISTS deployments (
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
  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
  updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
  started_at INTEGER,
  finished_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_deployments_app_created
  ON deployments(app_id, created_at DESC);

CREATE TABLE IF NOT EXISTS deployment_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  deployment_id INTEGER NOT NULL,
  log_level TEXT NOT NULL,
  message TEXT NOT NULL,
  created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX IF NOT EXISTS idx_deployment_logs_deployment
  ON deployment_logs(deployment_id, created_at ASC);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  app_id INTEGER NOT NULL,
  delivery_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  commit_sha TEXT,
  received_at INTEGER NOT NULL DEFAULT (unixepoch()),
  UNIQUE(app_id, delivery_id)
);
