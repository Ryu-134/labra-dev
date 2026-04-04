variable "name_prefix" {
  type = string
}

variable "state_bucket_name" {
  type = string
}

variable "lock_table_name" {
  type    = string
  default = null
}

variable "force_destroy" {
  type    = bool
  default = false
}

variable "tags" {
  type    = map(string)
  default = {}
}
