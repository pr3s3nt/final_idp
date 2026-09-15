# PostgreSQL on a kind cluster: its own namespace, StatefulSet, Service and an
# application role whose password rotates when password_revision increases.
terraform {
  required_providers {
    kubernetes = { source = "hashicorp/kubernetes", version = "~> 2.38" }
    random     = { source = "hashicorp/random", version = "~> 3.7" }
  }
}

variable "name" { type = string }
variable "storage_gb" { type = number }
variable "password_revision" { type = number }
variable "k8s_cluster" {
  description = "Outputs of the required k8s-cluster resource"
  type        = any
  sensitive   = true
}
variable "tags" {
  type    = map(string)
  default = {}
}

locals {
  kubeconfig = yamldecode(var.k8s_cluster.kubeconfig)
  namespace  = substr("res-${replace(lower(var.name), "/[^a-z0-9-]/", "-")}", 0, 60)
  labels     = { "app.kubernetes.io/name" = "postgres", "app.kubernetes.io/managed-by" = "idp" }
}

provider "kubernetes" {
  host                   = local.kubeconfig.clusters[0].cluster.server
  cluster_ca_certificate = base64decode(local.kubeconfig.clusters[0].cluster["certificate-authority-data"])
  client_certificate     = base64decode(local.kubeconfig.users[0].user["client-certificate-data"])
  client_key             = base64decode(local.kubeconfig.users[0].user["client-key-data"])
}

resource "random_password" "superuser" {
  length  = 32
  special = false
}

resource "random_password" "app" {
  length  = 32
  special = false
  keepers = { revision = var.password_revision }
}

resource "kubernetes_namespace_v1" "this" {
  metadata { name = local.namespace }
}

resource "kubernetes_secret_v1" "credentials" {
  metadata {
    name      = "postgres-credentials"
    namespace = kubernetes_namespace_v1.this.metadata[0].name
  }
  data = {
    superuser-password = random_password.superuser.result
    app-password       = random_password.app.result
  }
}

resource "kubernetes_service_v1" "this" {
  metadata {
    name      = "postgres"
    namespace = kubernetes_namespace_v1.this.metadata[0].name
  }
  spec {
    selector = local.labels
    port {
      port        = 5432
      target_port = 5432
    }
  }
}

resource "kubernetes_stateful_set_v1" "this" {
  metadata {
    name      = "postgres"
    namespace = kubernetes_namespace_v1.this.metadata[0].name
    labels    = local.labels
  }
  wait_for_rollout = true
  spec {
    service_name = kubernetes_service_v1.this.metadata[0].name
    replicas     = 1
    selector { match_labels = local.labels }
    template {
      metadata { labels = local.labels }
      spec {
        container {
          name  = "postgres"
          image = "postgres:17-alpine"
          env {
            name = "POSTGRES_PASSWORD"
            value_from {
              secret_key_ref {
                name = kubernetes_secret_v1.credentials.metadata[0].name
                key  = "superuser-password"
              }
            }
          }
          env {
            name  = "PGDATA"
            value = "/var/lib/postgresql/data/pgdata"
          }
          port { container_port = 5432 }
          readiness_probe {
            exec { command = ["pg_isready", "-U", "postgres"] }
            period_seconds = 3
          }
          volume_mount {
            name       = "data"
            mount_path = "/var/lib/postgresql/data"
          }
        }
      }
    }
    volume_claim_template {
      metadata { name = "data" }
      spec {
        access_modes = ["ReadWriteOnce"]
        resources { requests = { storage = "${var.storage_gb}Gi" } }
      }
    }
  }
}

# Creates the application database and role, and sets the role password. The
# job name carries the password revision so a rotation runs a new job.
resource "kubernetes_job_v1" "app_role" {
  depends_on = [kubernetes_stateful_set_v1.this]
  metadata {
    name      = "app-role-r${var.password_revision}"
    namespace = kubernetes_namespace_v1.this.metadata[0].name
  }
  wait_for_completion = true
  timeouts { create = "5m" }
  spec {
    backoff_limit = 10
    template {
      metadata {}
      spec {
        restart_policy = "OnFailure"
        container {
          name    = "psql"
          image   = "postgres:17-alpine"
          command = ["sh", "-c", <<-EOT
            set -e
            until pg_isready -h postgres -U postgres; do sleep 2; done
            psql -h postgres -U postgres -v ON_ERROR_STOP=1 -c "DO \$\$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'app') THEN CREATE ROLE app LOGIN; END IF; END \$\$;"
            psql -h postgres -U postgres -v ON_ERROR_STOP=1 -c "ALTER ROLE app PASSWORD '$APP_PASSWORD'"
            psql -h postgres -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'app'" | grep -q 1 || psql -h postgres -U postgres -c "CREATE DATABASE app OWNER app"
          EOT
          ]
          env {
            name = "PGPASSWORD"
            value_from {
              secret_key_ref {
                name = kubernetes_secret_v1.credentials.metadata[0].name
                key  = "superuser-password"
              }
            }
          }
          env {
            name = "APP_PASSWORD"
            value_from {
              secret_key_ref {
                name = kubernetes_secret_v1.credentials.metadata[0].name
                key  = "app-password"
              }
            }
          }
        }
      }
    }
  }
}

output "host" { value = "postgres.${local.namespace}.svc.cluster.local" }
output "port" { value = "5432" }
output "database" { value = "app" }
output "username" { value = "app" }
output "password" {
  value     = random_password.app.result
  sensitive = true
}
