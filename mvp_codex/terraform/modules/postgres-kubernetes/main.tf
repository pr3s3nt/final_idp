provider "kubernetes" {
  config_path    = var.kubeconfig_path
  config_context = var.kube_context
}

resource "kubernetes_service_v1" "postgres" {
  metadata {
    name      = var.service_name
    namespace = var.namespace
    labels = {
      "app.kubernetes.io/name"       = var.service_name
      "app.kubernetes.io/managed-by" = "idp-terraform"
    }
  }

  spec {
    cluster_ip = "None"
    selector = {
      "app.kubernetes.io/name" = var.service_name
    }
    port {
      name        = "postgres"
      port        = 5432
      target_port = 5432
    }
  }
}

resource "kubernetes_stateful_set_v1" "postgres" {
  metadata {
    name      = var.service_name
    namespace = var.namespace
    labels = {
      "app.kubernetes.io/name"       = var.service_name
      "app.kubernetes.io/managed-by" = "idp-terraform"
    }
  }

  spec {
    service_name = kubernetes_service_v1.postgres.metadata[0].name
    replicas     = 1

    selector {
      match_labels = {
        "app.kubernetes.io/name" = var.service_name
      }
    }

    template {
      metadata {
        labels = {
          "app.kubernetes.io/name"       = var.service_name
          "app.kubernetes.io/managed-by" = "idp-terraform"
        }
      }

      spec {
        container {
          name  = "postgres"
          image = var.postgres_image

          port {
            name           = "postgres"
            container_port = 5432
          }

          env {
            name  = "POSTGRES_DB"
            value = "notes"
          }
          env {
            name  = "POSTGRES_USER"
            value = "notes"
          }
          env {
            name = "POSTGRES_PASSWORD"
            value_from {
              secret_key_ref {
                name = var.db_secret_name
                key  = "password"
              }
            }
          }

          volume_mount {
            name       = "data"
            mount_path = "/var/lib/postgresql/data"
          }

          readiness_probe {
            exec {
              command = ["pg_isready", "-U", "notes", "-d", "notes"]
            }
            initial_delay_seconds = 5
            period_seconds        = 5
          }
        }
      }
    }

    volume_claim_template {
      metadata {
        name = "data"
      }
      spec {
        access_modes = ["ReadWriteOnce"]
        resources {
          requests = {
            storage = "${var.storage_mi}Mi"
          }
        }
      }
    }
  }
}
