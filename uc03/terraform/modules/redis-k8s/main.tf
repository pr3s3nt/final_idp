# Redis on a kind cluster in its own namespace.
terraform {
  required_providers {
    kubernetes = { source = "hashicorp/kubernetes", version = "~> 2.38" }
  }
}

variable "name" { type = string }
variable "k8s_cluster" {
  type      = any
  sensitive = true
}
variable "tags" {
  type    = map(string)
  default = {}
}

locals {
  kubeconfig = yamldecode(var.k8s_cluster.kubeconfig)
  namespace  = substr("res-${replace(lower(var.name), "/[^a-z0-9-]/", "-")}", 0, 60)
  labels     = { "app.kubernetes.io/name" = "redis", "app.kubernetes.io/managed-by" = "idp" }
}

provider "kubernetes" {
  host                   = local.kubeconfig.clusters[0].cluster.server
  cluster_ca_certificate = base64decode(local.kubeconfig.clusters[0].cluster["certificate-authority-data"])
  client_certificate     = base64decode(local.kubeconfig.users[0].user["client-certificate-data"])
  client_key             = base64decode(local.kubeconfig.users[0].user["client-key-data"])
}

resource "kubernetes_namespace_v1" "this" {
  metadata { name = local.namespace }
}

resource "kubernetes_deployment_v1" "this" {
  metadata {
    name      = "redis"
    namespace = kubernetes_namespace_v1.this.metadata[0].name
    labels    = local.labels
  }
  wait_for_rollout = true
  spec {
    replicas = 1
    selector { match_labels = local.labels }
    template {
      metadata { labels = local.labels }
      spec {
        container {
          name  = "redis"
          image = "redis:7-alpine"
          port { container_port = 6379 }
          readiness_probe {
            tcp_socket { port = 6379 }
            period_seconds = 3
          }
        }
      }
    }
  }
}

resource "kubernetes_service_v1" "this" {
  metadata {
    name      = "redis"
    namespace = kubernetes_namespace_v1.this.metadata[0].name
  }
  spec {
    selector = local.labels
    port {
      port        = 6379
      target_port = 6379
    }
  }
}

output "host" { value = "redis.${local.namespace}.svc.cluster.local" }
output "port" { value = "6379" }
