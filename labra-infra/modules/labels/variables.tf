#  keep project and env separate so naming stays readable and predictable for both of you
variable "project_name" {
  description = "Project identifier used in names and tags"
  type        = string
}

variable "environment" {
  description = "Environment identifier"
  type        = string
}

#  use component as a suffix when we need to split resources by concern
variable "component" {
  description = "Optional component suffix for resource names"
  type        = string
  default     = ""
}

#  keep owner explicit so accountability is obvious in AWS tags
variable "owner" {
  description = "Owner responsible for this stack"
  type        = string
}

variable "extra_tags" {
  description = "Optional additional tags"
  type        = map(string)
  default     = {}
}

#  keep roadmap markers in tags so when we debug in AWS we know exactly which milestone made a thing
variable "roadmap_phase" {
  description = "Roadmap phase label for tagging"
  type        = string
  default     = "Phase 4"
}

variable "roadmap_version" {
  description = "Roadmap version label for tagging"
  type        = string
  default     = "Ver 1.0"
}
