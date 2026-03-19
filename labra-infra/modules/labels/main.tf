locals {
  #  add this suffix only when component is set so names do not get awkward double dashes
  component_suffix = var.component != "" ? "-${var.component}" : ""

  #  use one canonical prefix so backend and frontend can map outputs to resource names quickly
  resource_prefix = "${var.project_name}-${var.environment}${local.component_suffix}"

  #  keep a shared base tag set so our AWS filtering is consistent everywhere
  base_tags = {
    Project      = var.project_name
    Environment  = var.environment
    Owner        = var.owner
    ManagedBy    = "Terraform"
    Version      = var.roadmap_version
    RoadmapPhase = var.roadmap_phase
  }

  #  merge optional extras last so we can extend tags without touching core logic
  tags = merge(local.base_tags, var.extra_tags)
}
