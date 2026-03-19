#  keep these core names here so we can reuse the same file shape across environments
variable "project_name" {
  description = "Project identifier used in resource names and tags"
  type        = string
  default     = "labra-infra"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "dev"
}

variable "component" {
  description = "Component suffix used by labels module"
  type        = string
  default     = "platform"
}

#  keep region and owner explicit so we can trace resources without guessing
variable "aws_region" {
  description = "AWS region for this environment"
  type        = string
  default     = "us-west-2"
}

variable "owner" {
  description = "Owner tag value"
  type        = string
  default     = "cpsc465-infra"
}

variable "extra_tags" {
  description = "Additional tags to apply to all resources"
  type        = map(string)
  default     = {}
}

#  keep roadmap markers on resources so when either of you checks AWS you can map infra to milestone fast
variable "roadmap_phase" {
  description = "Roadmap phase label applied as a resource tag"
  type        = string
  default     = "Phase 4"
}

variable "roadmap_version" {
  description = "Roadmap version label applied as a resource tag"
  type        = string
  default     = "Ver 1.0"
}

#  use this toggle only for the one-time backend bootstrap apply
variable "bootstrap_state_backend" {
  description = "When true create S3 and DynamoDB backend resources"
  type        = bool
  default     = false
}

variable "state_bucket_name" {
  description = "Globally unique S3 bucket name for Terraform state"
  type        = string
}

variable "state_lock_table_name" {
  description = "Optional DynamoDB lock table name"
  type        = string
  default     = null
}

variable "state_bucket_force_destroy" {
  description = "Whether the state bucket can be force-destroyed"
  type        = bool
  default     = false
}

#  keep app contract fields here because backend/frontend both care about this identity
variable "app_name" {
  description = "Logical app name tag used by runtime resources"
  type        = string
  default     = "demo-app"
}

variable "build_type" {
  description = "Build type for this environment"
  type        = string
  default     = "static"

  validation {
    condition     = var.build_type == "static"
    error_message = "Phase 4 scope only supports build_type = static"
  }
}

#  kept only the static knobs we actually need through Phase 4
variable "static_site_bucket_name" {
  description = "Optional explicit static site bucket name override"
  type        = string
  default     = null
}

variable "static_default_root_object" {
  description = "Default CloudFront root object for static app"
  type        = string
  default     = "index.html"
}

variable "static_enable_spa_routing" {
  description = "Serve index.html for 403 and 404 to support SPA routes"
  type        = bool
  default     = true
}

variable "static_price_class" {
  description = "CloudFront price class for static runtime"
  type        = string
  default     = "PriceClass_100"
}

variable "static_force_destroy" {
  description = "Allow force destroy for static bucket during teardown"
  type        = bool
  default     = false
}

variable "static_release_prefix" {
  description = "S3 prefix for versioned static release artifacts"
  type        = string
  default     = "releases/"
}

variable "static_release_retention_days" {
  description = "Retention in days for current static release objects"
  type        = number
  default     = 90
}

variable "static_noncurrent_retention_days" {
  description = "Retention in days for noncurrent static release object versions"
  type        = number
  default     = 30
}

variable "static_enable_alarms" {
  description = "Create baseline CloudFront alarms for static runtime"
  type        = bool
  default     = true
}

variable "static_alarm_period_seconds" {
  description = "CloudWatch period for static alarms"
  type        = number
  default     = 300
}

variable "static_alarm_evaluation_periods" {
  description = "CloudWatch evaluation periods for static alarms"
  type        = number
  default     = 1
}

variable "static_cf_5xx_rate_threshold" {
  description = "CloudFront 5xx error rate threshold"
  type        = number
  default     = 1
}
