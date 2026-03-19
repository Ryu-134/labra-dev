#  kept these inputs tight so this module only does what we need for static deploys through Phase 4
variable "name_prefix" {
  description = "Prefix used to name static hosting resources"
  type        = string
}

variable "app_name" {
  description = "Logical app name for static deploy resources"
  type        = string
}

#  allow override here but if we leave it null  derive a deterministic name from prefix + account id
variable "bucket_name" {
  description = "Optional explicit bucket name for static site assets"
  type        = string
  default     = null
}

variable "default_root_object" {
  description = "Default object served by CloudFront"
  type        = string
  default     = "index.html"
}

variable "price_class" {
  description = "CloudFront price class for cost control"
  type        = string
  default     = "PriceClass_100"
}

variable "enable_spa_routing" {
  description = "Whether 403 and 404 should resolve to index.html for SPA routing"
  type        = bool
  default     = true
}

variable "force_destroy" {
  description = "Whether static bucket objects can be force-deleted with terraform destroy"
  type        = bool
  default     = false
}

#  kept release and retention knobs because backend deploy history depends on them
variable "release_prefix" {
  description = "S3 prefix where per-release artifacts are stored"
  type        = string
  default     = "releases/"
}

variable "release_retention_days" {
  description = "Retention in days for current release objects"
  type        = number
  default     = 90
}

variable "noncurrent_retention_days" {
  description = "Retention in days for noncurrent versioned release objects"
  type        = number
  default     = 30
}

#  keep alarms minimal here on purpose so we get useful signal without noisy extras
variable "enable_alarms" {
  description = "Create baseline CloudFront alarms for static runtime"
  type        = bool
  default     = true
}

variable "alarm_period_seconds" {
  description = "CloudWatch alarm period in seconds"
  type        = number
  default     = 300
}

variable "alarm_evaluation_periods" {
  description = "CloudWatch alarm evaluation periods"
  type        = number
  default     = 1
}

variable "cf_5xx_rate_threshold" {
  description = "CloudFront 5xx error rate threshold percentage"
  type        = number
  default     = 1
}

variable "tags" {
  description = "Tags applied to static runtime resources"
  type        = map(string)
  default     = {}
}
