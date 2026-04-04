variable "project_name" {
  type    = string
  default = "labra-infra"
}

variable "environment" {
  type    = string
  default = "dev"
}

variable "component" {
  type    = string
  default = "platform"
}

variable "aws_region" {
  type    = string
  default = "us-west-2"

  validation {
    condition     = can(regex("^[a-z]{2}(-[a-z]+)+-[0-9]+$", var.aws_region))
    error_message = "aws_region must look like us-west-2."
  }
}

variable "owner" {
  type    = string
  default = "cpsc465-infra"
}

variable "extra_tags" {
  type    = map(string)
  default = {}
}

variable "roadmap_phase" {
  type    = string
  default = "Phase 4"
}

variable "roadmap_version" {
  type    = string
  default = "Ver 1.0"
}

variable "bootstrap_state_backend" {
  type    = bool
  default = false
}

variable "state_bucket_name" {
  type = string
}

variable "state_lock_table_name" {
  type    = string
  default = null
}

variable "state_bucket_force_destroy" {
  type    = bool
  default = false
}

variable "app_name" {
  type    = string
  default = "demo-app"
}

variable "build_type" {
  type    = string
  default = "static"
}

variable "static_site_bucket_name" {
  type    = string
  default = null
}

variable "static_default_root_object" {
  type    = string
  default = "index.html"
}

variable "static_enable_spa_routing" {
  type    = bool
  default = true
}

variable "static_price_class" {
  type    = string
  default = "PriceClass_100"
}

variable "static_force_destroy" {
  type    = bool
  default = false
}

variable "static_release_prefix" {
  type    = string
  default = "releases/"
}

variable "static_release_retention_days" {
  type    = number
  default = 90
}

variable "static_noncurrent_retention_days" {
  type    = number
  default = 30
}

variable "static_enable_alarms" {
  type    = bool
  default = true
}

variable "static_alarm_period_seconds" {
  type    = number
  default = 300

  validation {
    condition     = var.static_alarm_period_seconds > 0
    error_message = "static_alarm_period_seconds must be > 0."
  }
}

variable "static_alarm_evaluation_periods" {
  type    = number
  default = 1
}

variable "static_cf_5xx_rate_threshold" {
  type    = number
  default = 1
}

variable "runner_enabled" {
  type    = bool
  default = false
}

variable "runner_launch_type" {
  type    = string
  default = "FARGATE"

  validation {
    condition     = var.runner_launch_type == "FARGATE"
    error_message = "runner_launch_type must be FARGATE."
  }
}

variable "runner_task_cpu" {
  type    = number
  default = 1024
}

variable "runner_task_memory" {
  type    = number
  default = 2048
}

variable "runner_ephemeral_storage_gib" {
  type    = number
  default = 21
}

variable "runner_timeout_seconds" {
  type    = number
  default = 3600

  validation {
    condition     = var.runner_timeout_seconds > 0
    error_message = "runner_timeout_seconds must be > 0."
  }
}

variable "runner_container_image" {
  type    = string
  default = "public.ecr.aws/docker/library/node:20-alpine"

  validation {
    condition     = length(trimspace(var.runner_container_image)) > 0
    error_message = "runner_container_image cannot be empty."
  }
}

variable "runner_assign_public_ip" {
  type    = bool
  default = false
}

variable "runner_subnet_ids" {
  type    = list(string)
  default = []
}

variable "runner_security_group_ids" {
  type    = list(string)
  default = []
}

variable "runner_log_retention_days" {
  type    = number
  default = 14
}

variable "runner_execution_role_name" {
  type    = string
  default = "labra-runner-execution-role"
}

variable "runner_task_role_name" {
  type    = string
  default = "labra-runner-task-role"
}
