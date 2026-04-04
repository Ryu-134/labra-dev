variable "name_prefix" {
  type = string
}

variable "app_name" {
  type = string
}

variable "build_type" {
  type    = string
  default = "static"

  validation {
    condition     = var.build_type == "static"
    error_message = "build_type must be static."
  }
}

variable "region" {
  type    = string
  default = null
}

variable "bucket_name" {
  type    = string
  default = null
}

variable "default_root_object" {
  type    = string
  default = "index.html"
}

variable "price_class" {
  type    = string
  default = "PriceClass_100"

  validation {
    condition     = contains(["PriceClass_100", "PriceClass_200", "PriceClass_All"], var.price_class)
    error_message = "price_class must be a valid CloudFront price class."
  }
}

variable "enable_spa_routing" {
  type    = bool
  default = true
}

variable "force_destroy" {
  type    = bool
  default = false
}

variable "release_prefix" {
  type    = string
  default = "releases/"

  validation {
    condition     = length(trimspace(var.release_prefix)) > 0 && endswith(var.release_prefix, "/")
    error_message = "release_prefix must be non-empty and end with '/'."
  }
}

variable "release_retention_days" {
  type    = number
  default = 90
}

variable "noncurrent_retention_days" {
  type    = number
  default = 30

  validation {
    condition     = var.noncurrent_retention_days > 0 && var.noncurrent_retention_days <= var.release_retention_days
    error_message = "noncurrent_retention_days must be > 0 and <= release_retention_days."
  }
}

variable "enable_alarms" {
  type    = bool
  default = true
}

variable "alarm_period_seconds" {
  type    = number
  default = 300
}

variable "alarm_evaluation_periods" {
  type    = number
  default = 1

  validation {
    condition     = var.alarm_evaluation_periods > 0
    error_message = "alarm_evaluation_periods must be > 0."
  }
}

variable "cf_5xx_rate_threshold" {
  type    = number
  default = 1

  validation {
    condition     = var.cf_5xx_rate_threshold >= 0 && var.cf_5xx_rate_threshold <= 100
    error_message = "cf_5xx_rate_threshold must be between 0 and 100."
  }
}

variable "tags" {
  type    = map(string)
  default = {}
}
