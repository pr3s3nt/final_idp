output "host" {
  value = "${kubernetes_service_v1.postgres.metadata[0].name}.${var.namespace}.svc.cluster.local"
}

output "port" {
  value = 5432
}

output "database" {
  value = "notes"
}

output "username" {
  value = "notes"
}

output "sslmode" {
  value = "disable"
}

output "infrastructure_reference" {
  value = "kubernetes://${var.kube_context}/namespaces/${var.namespace}/statefulsets/${kubernetes_stateful_set_v1.postgres.metadata[0].name}"
}
