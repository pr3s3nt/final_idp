# k8s-cluster on target kind-local: a kind cluster for one application +
# environment, wired to the local image registry, with Argo CD installed.
terraform {
  required_providers {
    kind = { source = "tehcyx/kind", version = "~> 0.11" }
    helm = { source = "hashicorp/helm", version = "~> 3.3" }
    null = { source = "hashicorp/null", version = "~> 3.2" }
  }
}

variable "name" { type = string }
variable "node_count" { type = number }
variable "node_image" { type = string }
variable "argocd_chart_version" { type = string }
variable "image_registry_mirror" { type = string }
variable "registry_container" { type = string }
variable "tags" {
  type    = map(string)
  default = {}
}

locals {
  cluster_name = substr(replace(lower(var.name), "/[^a-z0-9-]/", "-"), 0, 40)
  workers      = var.node_count - 1
}

resource "kind_cluster" "this" {
  name           = local.cluster_name
  node_image     = var.node_image
  wait_for_ready = true

  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"
    containerd_config_patches = [<<-TOML
      [plugins."io.containerd.cri.v1.images".registry]
        config_path = "/etc/containerd/certs.d"
    TOML
    ]
    node {
      role = "control-plane"
    }
    dynamic "node" {
      for_each = range(local.workers)
      content {
        role = "worker"
      }
    }
  }
}

# Point containerd on every node at the registry container for the mirror host.
resource "null_resource" "registry_mirror" {
  triggers = {
    cluster  = kind_cluster.this.id
    mirror   = var.image_registry_mirror
    registry = var.registry_container
  }
  provisioner "local-exec" {
    interpreter = ["bash", "-c"]
    command     = <<-EOT
      set -e
      for node in $(kind get nodes --name ${local.cluster_name}); do
        docker exec "$node" mkdir -p /etc/containerd/certs.d/${var.image_registry_mirror}
        printf '[host."http://${var.registry_container}:5000"]\n  capabilities = ["pull", "resolve"]\n' |
          docker exec -i "$node" tee /etc/containerd/certs.d/${var.image_registry_mirror}/hosts.toml >/dev/null
      done
    EOT
  }
}

provider "helm" {
  kubernetes = {
    host                   = kind_cluster.this.endpoint
    client_certificate     = kind_cluster.this.client_certificate
    client_key             = kind_cluster.this.client_key
    cluster_ca_certificate = kind_cluster.this.cluster_ca_certificate
  }
}

resource "helm_release" "argocd" {
  depends_on       = [null_resource.registry_mirror]
  name             = "argocd"
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  version          = var.argocd_chart_version
  namespace        = "argocd"
  create_namespace = true
  wait             = true
  timeout          = 900
  values = [yamlencode({
    dex           = { enabled = false }
    notifications = { enabled = false }
    applicationSet = { replicas = 0 }
    configs = {
      cm     = { "timeout.reconciliation" = "20s" }
      params = { "server.insecure" = true }
    }
  })]
}

output "cluster_name" { value = local.cluster_name }
output "cluster_kind" { value = "kind" }
output "image_registry_mirror" { value = var.image_registry_mirror }
output "kubeconfig" {
  value     = kind_cluster.this.kubeconfig
  sensitive = true
}
