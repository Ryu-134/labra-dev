output "resource_prefix" {
  description = "Canonical name prefix for resources."
  value       = local.resource_prefix
}

output "tags" {
  description = "Merged tag map for downstream modules."
  value       = local.tags
}
