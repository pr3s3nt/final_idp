variable "aws_region" {
  type    = string
  default = "ap-southeast-1"
}
variable "expected_account_id" {
  type = string

  validation {
    condition     = can(regex("^[0-9]{12}$", var.expected_account_id))
    error_message = "expected_account_id must be a 12-digit AWS account ID."
  }
}

variable "cluster_version" {
  type    = string
  default = "1.35"
}

variable "node_instance_type" {
  type    = string
  default = "t3.small"
}

variable "cluster_public_access_cidrs" {
  type = list(string)

  validation {
    condition     = length(var.cluster_public_access_cidrs) > 0
    error_message = "At least one explicit EKS API access CIDR is required."
  }
}
