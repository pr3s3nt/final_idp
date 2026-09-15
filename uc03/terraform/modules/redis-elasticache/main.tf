# Redis on target aws: single-node ElastiCache in the network's private subnets.
terraform {
  required_providers {
    aws = { source = "hashicorp/aws", version = "~> 6.0" }
  }
}

variable "name" { type = string }
variable "region" { type = string }
variable "node_type" { type = string }
variable "tags" { type = map(string) }
variable "network" { type = any }

provider "aws" {
  region = var.region
  default_tags { tags = var.tags }
}

resource "aws_elasticache_cluster" "this" {
  cluster_id           = substr(replace(var.name, "--", "-"), 0, 40)
  engine               = "redis"
  engine_version       = "7.1"
  node_type            = var.node_type
  num_cache_nodes      = 1
  parameter_group_name = "default.redis7"
  port                 = 6379
  subnet_group_name    = var.network.cache_subnet_group
  security_group_ids   = [var.network.data_security_group_id]
  apply_immediately    = true
}

output "host" { value = aws_elasticache_cluster.this.cache_nodes[0].address }
output "port" { value = "6379" }
