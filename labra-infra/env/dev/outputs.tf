#  expose this so both of you can line up naming with what AWS actually has
output "resource_prefix" {
  description = "Computed name prefix used by this environment"
  value       = module.labels.resource_prefix
}

#  expose final tags so we can quickly sanity check ownership and phase metadata
output "tags" {
  description = "Merged tags used in this environment"
  value       = module.labels.tags
}

#  keep these visible so frontend/backend can verify we are on the expected roadmap target
output "roadmap_phase" {
  description = "Roadmap phase marker"
  value       = var.roadmap_phase
}

output "roadmap_version" {
  description = "Roadmap version marker"
  value       = var.roadmap_version
}

#  keep app contract fields visible since backend/frontend both read them
output "app_name" {
  description = "Logical app name configured for this environment"
  value       = var.app_name
}

output "build_type" {
  description = "Configured build type for this environment"
  value       = var.build_type
}

#  return these only when bootstrap is enabled
output "state_bucket_name" {
  description = "Terraform state bucket name when bootstrap_state_backend is true"
  value       = try(module.state_bootstrap[0].state_bucket_name, null)
}

output "state_lock_table_name" {
  description = "Terraform lock table name when bootstrap_state_backend is true"
  value       = try(module.state_bootstrap[0].lock_table_name, null)
}

#  expect backend deploy logic to use these static runtime outputs directly
output "static_bucket_name" {
  description = "S3 bucket name for static deploy artifacts"
  value       = module.static_runtime.bucket_name
}

output "static_distribution_id" {
  description = "CloudFront distribution ID for static deploy path"
  value       = module.static_runtime.distribution_id
}

output "static_site_url" {
  description = "Public URL for static deploy mode"
  value       = module.static_runtime.site_url
}

output "static_release_prefix" {
  description = "Release artifact prefix used by static deploy flow"
  value       = module.static_runtime.release_prefix
}

output "static_alarm_names" {
  description = "CloudFront alarm names for static runtime"
  value       = module.static_runtime.alarm_names
}
