#  keep these input notes here so whoever is wiring envs does not have to guess intent
variable "name_prefix" {
  description = "Name prefix used for tagging and defaults"
  type        = string
}

variable "state_bucket_name" {
  description = "Globally unique S3 bucket name for Terraform state"
  type        = string
}

variable "lock_table_name" {
  description = "Optional lock table name defaults to <name_prefix>-terraform-locks"
  type        = string
  default     = null
}

#  keep this false by default because deleting a state bucket by accident is brutal
variable "force_destroy" {
  description = "Whether the S3 state bucket can be destroyed even if non-empty"
  type        = bool
  default     = false
}

variable "tags" {
  description = "Tags applied to all resources in this module"
  type        = map(string)
  default     = {}
}
