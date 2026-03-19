terraform {
  #  keep this aligned with root constraints so nobody gets odd version mismatch errors
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  #  keep local state first until one of us runs the backend bootstrap flow
}

locals {
  #  build provider tags from vars so frontend/backend can see clean ownership and phase metadata in AWS
  provider_default_tags = merge(
    {
      Project      = var.project_name
      Environment  = var.environment
      Owner        = var.owner
      ManagedBy    = "Terraform"
      Version      = var.roadmap_version
      RoadmapPhase = var.roadmap_phase
    },
    var.extra_tags
  )
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = local.provider_default_tags
  }
}

#  keep labels central so both of you can trust naming/tag patterns in outputs and logs
module "labels" {
  source = "../../modules/labels"

  project_name    = var.project_name
  environment     = var.environment
  component       = var.component
  owner           = var.owner
  extra_tags      = var.extra_tags
  roadmap_phase   = var.roadmap_phase
  roadmap_version = var.roadmap_version
}

#  only use this when we need to create shared remote state infra for the first time
module "state_bootstrap" {
  count  = var.bootstrap_state_backend ? 1 : 0
  source = "../../modules/state-bootstrap"

  name_prefix       = module.labels.resource_prefix
  state_bucket_name = var.state_bucket_name
  lock_table_name   = var.state_lock_table_name
  force_destroy     = var.state_bucket_force_destroy
  tags              = module.labels.tags
}

#  kept this focused on what we need through Phase 4 for static deploys
module "static_runtime" {
  source = "../../modules/static_runtime"

  name_prefix               = module.labels.resource_prefix
  app_name                  = var.app_name
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
  tags                      = module.labels.tags
}
