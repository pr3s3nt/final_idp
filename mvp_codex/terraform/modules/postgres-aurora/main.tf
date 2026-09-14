provider "aws" {
  region = var.aws_region
}

locals {
  compact_resource_id = replace(lower(var.resource_instance_id), "-", "")
  cluster_identifier  = "idp-${substr(local.compact_resource_id, 0, 24)}"
  ownership_tags = merge(var.tags, {
    "idp.managed-by"           = "mvp-codex"
    "idp.resource-instance-id" = var.resource_instance_id
  })
}

resource "aws_rds_cluster" "postgres" {
  cluster_identifier = local.cluster_identifier
  engine             = "aurora-postgresql"
  engine_mode        = "provisioned"
  engine_version     = var.engine_version

  database_name   = var.database_name
  master_username = var.master_username

  manage_master_user_password = true
  storage_encrypted           = true
  db_subnet_group_name        = var.db_subnet_group_name
  vpc_security_group_ids      = var.vpc_security_group_ids
  network_type                = "IPV4"
  port                        = 5432
  deletion_protection         = false
  skip_final_snapshot         = true
  copy_tags_to_snapshot       = true

  serverlessv2_scaling_configuration {
    min_capacity = var.min_capacity
    max_capacity = var.max_capacity
  }

  tags = local.ownership_tags
}

resource "aws_rds_cluster_instance" "writer" {
  identifier          = "${local.cluster_identifier}-writer"
  cluster_identifier  = aws_rds_cluster.postgres.id
  instance_class      = "db.serverless"
  engine              = aws_rds_cluster.postgres.engine
  engine_version      = aws_rds_cluster.postgres.engine_version
  publicly_accessible = false

  tags = local.ownership_tags
}
