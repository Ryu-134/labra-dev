output "resource_prefix" {
  value = local.resource_prefix
}

output "tags" {
  value = local.tags
}

output "roadmap_phase" {
  value = var.roadmap_phase
}

output "roadmap_version" {
  value = var.roadmap_version
}

output "app_name" {
  value = var.app_name
}

output "build_type" {
  value = var.build_type
}

output "state_bucket_name" {
  value = try(module.state_bootstrap[0].state_bucket_name, null)
}

output "state_lock_table_name" {
  value = try(module.state_bootstrap[0].lock_table_name, null)
}

output "static_bucket_name" {
  value = module.static_runtime.bucket_name
}

output "static_distribution_id" {
  value = module.static_runtime.distribution_id
}

output "static_site_url" {
  value = module.static_runtime.site_url
}

output "static_release_prefix" {
  value = module.static_runtime.release_prefix
}

output "static_alarm_names" {
  value = module.static_runtime.alarm_names
}

output "runner_contract" {
  value = {
    enabled                 = var.runner_enabled
    launch_type             = var.runner_launch_type
    region                  = var.aws_region
    container_image         = var.runner_container_image
    timeout_seconds         = var.runner_timeout_seconds
    ephemeral_storage_gib   = var.runner_ephemeral_storage_gib
    assign_public_ip        = var.runner_assign_public_ip
    subnet_ids              = var.runner_subnet_ids
    security_group_ids      = var.runner_security_group_ids
    task_cpu                = var.runner_task_cpu
    task_memory             = var.runner_task_memory
    log_retention_days      = var.runner_log_retention_days
    execution_role_name     = var.runner_execution_role_name
    task_role_name          = var.runner_task_role_name
    contract_schema_version = "1.0"
  }
}
