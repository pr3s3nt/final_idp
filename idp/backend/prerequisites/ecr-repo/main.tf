# CI registry stand-in on AWS: ECR repositories the demo images are pushed to.
# Not part of any deployment graph; created before and deleted after the AWS
# verification.
terraform {
  required_providers {
    aws = { source = "hashicorp/aws", version = "~> 6.0" }
  }
}

variable "region" {
  type    = string
  default = "ap-southeast-1"
}

provider "aws" {
  region = var.region
  default_tags { tags = { "idp-uc03-run" = "idp-uc03", "managed-by" = "terraform", "layer" = "ci-registry" } }
}

resource "aws_ecr_repository" "demo" {
  for_each             = toset(["shop-backend", "shop-frontend", "shop-worker"])
  name                 = each.value
  force_delete         = true
  image_tag_mutability = "MUTABLE"
}

data "aws_caller_identity" "current" {}

output "registry" {
  value = "${data.aws_caller_identity.current.account_id}.dkr.ecr.${var.region}.amazonaws.com"
}
