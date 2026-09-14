variable "aws_region" {
  type = string
}

variable "resource_instance_id" {
  type = string

  validation {
    condition     = can(regex("^[0-9a-fA-F-]{36}$", var.resource_instance_id))
    error_message = "resource_instance_id must be a UUID-shaped IDP resource identity."
  }
}

variable "db_subnet_group_name" {
  type = string
}

variable "vpc_security_group_ids" {
  type = list(string)

  validation {
    condition     = length(var.vpc_security_group_ids) > 0 && alltrue([for id in var.vpc_security_group_ids : can(regex("^sg-[0-9a-f]+$", id))])
    error_message = "vpc_security_group_ids must contain at least one AWS security-group ID."
  }
}

variable "database_name" {
  type    = string
  default = "notes"

  validation {
    condition     = can(regex("^[A-Za-z][A-Za-z0-9_]{0,62}$", var.database_name))
    error_message = "database_name must be a valid PostgreSQL database identifier."
  }
}

variable "master_username" {
  type    = string
  default = "notes_admin"
}

variable "engine_version" {
  type     = string
  default  = null
  nullable = true
}

variable "min_capacity" {
  type    = number
  default = 0.5

  validation {
    condition     = var.min_capacity >= 0 && var.min_capacity * 2 == floor(var.min_capacity * 2)
    error_message = "min_capacity must be a non-negative multiple of 0.5 ACU."
  }
}

variable "max_capacity" {
  type    = number
  default = 1

  validation {
    condition     = var.max_capacity >= var.min_capacity && var.max_capacity * 2 == floor(var.max_capacity * 2)
    error_message = "max_capacity must be a multiple of 0.5 ACU and at least min_capacity."
  }
}

variable "tags" {
  type    = map(string)
  default = {}
}
