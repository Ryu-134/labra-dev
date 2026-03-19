#  keep these core values obvious so we can spot bad env drift fast
project_name    = "labra-infra"
environment     = "dev"
component       = "platform"
owner           = "cpsc465-infra"
aws_region      = "us-west-2"
roadmap_phase   = "Phase 4"
roadmap_version = "Ver 1.0"

#  left these extra tags for class filtering and quick resource search
extra_tags = {
  Course = "CPSC465"
  Scope  = "Infrastructure"
}

#  flip this true only for the one bootstrap apply
bootstrap_state_backend = false

#  need a globally unique S3 bucket name here before first apply
state_bucket_name          = "replace-me-labra-infra-dev-tfstate"
state_lock_table_name      = "labra-infra-dev-platform-terraform-locks"
state_bucket_force_destroy = false

#  keep app contract values here so backend/frontend have one obvious source
app_name   = "demo-app"
build_type = "static"

#  kept only active static runtime settings needed through Phase 4
static_site_bucket_name          = null
static_default_root_object       = "index.html"
static_enable_spa_routing        = true
static_price_class               = "PriceClass_100"
static_force_destroy             = false
static_release_prefix            = "releases/"
static_release_retention_days    = 90
static_noncurrent_retention_days = 30
static_enable_alarms             = true
static_alarm_period_seconds      = 300
static_alarm_evaluation_periods  = 1
static_cf_5xx_rate_threshold     = 1
