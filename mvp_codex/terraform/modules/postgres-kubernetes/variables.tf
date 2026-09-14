variable "kubeconfig_path" {
  type      = string
  sensitive = true
}

variable "kube_context" {
  type = string
}

variable "namespace" {
  type = string
}

variable "service_name" {
  type = string
}

variable "storage_mi" {
  type = number

  validation {
    condition     = var.storage_mi >= 512 && floor(var.storage_mi) == var.storage_mi
    error_message = "storage_mi must be an integer of at least 512 MiB."
  }
}

variable "db_secret_name" {
  type = string
}

variable "postgres_image" {
  type = string

  validation {
    condition     = can(regex("@sha256:[0-9a-f]{64}$", var.postgres_image))
    error_message = "postgres_image must be pinned by sha256 digest."
  }
}
