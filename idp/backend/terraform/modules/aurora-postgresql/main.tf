# PostgreSQL on target aws: Aurora PostgreSQL Serverless v2 in the network's
# private subnets. Increasing password_revision rotates the master password.
terraform {
  required_providers {
    aws    = { source = "hashicorp/aws", version = "~> 6.0" }
    random = { source = "hashicorp/random", version = "~> 3.7" }
  }
}

variable "name" { type = string }
variable "region" { type = string }
variable "engine_version" { type = string }
variable "min_acu" { type = number }
variable "max_acu" { type = number }
variable "password_revision" { type = number }
variable "tags" { type = map(string) }
variable "network" { type = any }

provider "aws" {
  region = var.region
  default_tags { tags = var.tags }
}

resource "random_password" "master" {
  length  = 32
  special = false
  keepers = { revision = var.password_revision }
}

resource "aws_rds_cluster" "this" {
  cluster_identifier      = var.name
  engine                  = "aurora-postgresql"
  engine_mode             = "provisioned"
  engine_version          = var.engine_version
  database_name           = "app"
  master_username         = "app"
  master_password         = random_password.master.result
  db_subnet_group_name    = var.network.db_subnet_group
  vpc_security_group_ids  = [var.network.data_security_group_id]
  storage_encrypted       = true
  skip_final_snapshot     = true
  deletion_protection     = false
  backup_retention_period = 1
  apply_immediately       = true
  serverlessv2_scaling_configuration {
    min_capacity = var.min_acu
    max_capacity = var.max_acu
  }
}

resource "aws_rds_cluster_instance" "writer" {
  identifier          = "${var.name}-writer"
  cluster_identifier  = aws_rds_cluster.this.id
  instance_class      = "db.serverless"
  engine              = aws_rds_cluster.this.engine
  engine_version      = aws_rds_cluster.this.engine_version
  publicly_accessible = false
  apply_immediately   = true
}

output "host" { value = aws_rds_cluster.this.endpoint }
output "port" { value = tostring(aws_rds_cluster.this.port) }
output "database" { value = aws_rds_cluster.this.database_name }
output "username" { value = aws_rds_cluster.this.master_username }
output "password" {
  value     = random_password.master.result
  sensitive = true
}
