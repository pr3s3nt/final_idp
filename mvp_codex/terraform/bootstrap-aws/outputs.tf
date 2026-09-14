output "account_id" {
  value = data.aws_caller_identity.current.account_id
}

output "aws_region" {
  value = var.aws_region
}

output "cluster_name" {
  value = aws_eks_cluster.main.name
}

output "cluster_arn" {
  value = aws_eks_cluster.main.arn
}

output "cluster_endpoint" {
  value = aws_eks_cluster.main.endpoint
}

output "cluster_ca_data" {
  value = aws_eks_cluster.main.certificate_authority[0].data
}

output "node_group_name" {
  value = aws_eks_node_group.main.node_group_name
}

output "node_instance_type" {
  value = var.node_instance_type
}

output "vpc_id" {
  value = aws_vpc.main.id
}

output "db_subnet_group_name" {
  value = aws_db_subnet_group.aurora.name
}

output "aurora_security_group_id" {
  value = aws_security_group.aurora.id
}

output "external_secrets_role_arn" {
  value = aws_iam_role.external_secrets.arn
}

output "ecr_repositories" {
  value = { for name, repository in aws_ecr_repository.repository : name => repository.repository_url }
}

output "ownership_tags" {
  value = local.ownership_tags
}
