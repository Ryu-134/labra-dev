terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = local.tags
  }
}

locals {
  component_suffix = var.component == "" ? "" : "-${var.component}"
  resource_prefix  = "${var.project_name}-${var.environment}${local.component_suffix}"
  tags = merge({
    Project      = var.project_name
    Environment  = var.environment
    Owner        = var.owner
    ManagedBy    = "Terraform"
    Version      = var.roadmap_version
    RoadmapPhase = var.roadmap_phase
  }, var.extra_tags)
}

module "state_bootstrap" {
  count  = var.bootstrap_state_backend ? 1 : 0
  source = "../../modules/state-bootstrap"

  name_prefix       = local.resource_prefix
  state_bucket_name = var.state_bucket_name
  lock_table_name   = var.state_lock_table_name
  force_destroy     = var.state_bucket_force_destroy
  tags              = local.tags
}

module "static_runtime" {
  source = "../../modules/static_runtime"

  name_prefix               = local.resource_prefix
  app_name                  = var.app_name
  build_type                = var.build_type
  region                    = var.aws_region
  bucket_name               = var.static_site_bucket_name
  default_root_object       = var.static_default_root_object
  price_class               = var.static_price_class
  enable_spa_routing        = var.static_enable_spa_routing
  force_destroy             = var.static_force_destroy
  release_prefix            = var.static_release_prefix
  release_retention_days    = var.static_release_retention_days
  noncurrent_retention_days = var.static_noncurrent_retention_days
  enable_alarms             = var.static_enable_alarms
  alarm_period_seconds      = var.static_alarm_period_seconds
  alarm_evaluation_periods  = var.static_alarm_evaluation_periods
  cf_5xx_rate_threshold     = var.static_cf_5xx_rate_threshold
  tags                      = local.tags
}
