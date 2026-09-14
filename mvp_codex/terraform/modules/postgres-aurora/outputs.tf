output "host" {
  value = aws_rds_cluster.postgres.endpoint
}

output "port" {
  value = aws_rds_cluster.postgres.port
}

output "database" {
  value = aws_rds_cluster.postgres.database_name
}

output "username" {
  value = aws_rds_cluster.postgres.master_username
}

output "sslmode" {
  value = "require"
}

# This is a reference only. The secret value is never read by this module.
output "credential_secret_arn" {
  value = aws_rds_cluster.postgres.master_user_secret[0].secret_arn
}

output "infrastructure_reference" {
  value = aws_rds_cluster.postgres.arn
}
