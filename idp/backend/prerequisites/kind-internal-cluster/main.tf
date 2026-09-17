# Platform-owned internal Kubernetes cluster for target kind-local, created once
# by prerequisites/kind-internal-cluster.sh before any deployment: a kind
# cluster wired to the local image registry, with both Argo CD and Fleet installed. The IDP
# never creates or destroys it; the catalog definition kind-internal-cluster
# (EXISTING) points to its connection record in the Secret Store.
terraform {
  required_providers {
    kind = { source = "tehcyx/kind", version = "~> 0.11" }
    helm = { source = "hashicorp/helm", version = "~> 3.3" }
    null = { source = "hashicorp/null", version = "~> 3.2" }
  }
}

variable "name" {
  type    = string
  default = "idp-internal"
}
variable "node_count" {
  type    = number
  default = 1
}
variable "node_image" {
  type    = string
  default = "kindest/node:v1.36.1"
}
variable "argocd_chart_version" {
  type    = string
  default = "10.9.1"
}
variable "image_registry_mirror" {
  type    = string
  default = "localhost:5055"
}
variable "registry_container" {
  type    = string
  default = "idp-uc03-registry"
}
variable "tags" {
  type    = map(string)
  default = {}
}

locals {
  cluster_name = substr(replace(lower(var.name), "/[^a-z0-9-]/", "-"), 0, 40)
  workers      = var.node_count - 1
}

variable "fleet_chart_version" {
  description = "Fleet chart version; fleet-crd and fleet are released together."
  type        = string
  default     = "0.16.1"
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

# Both CD systems run side by side (design decision 14): the IDP picks one per
# deployment through IDP_CD_PROVIDER, which is how the CD abstraction is shown
# to be replaceable rather than only claimed to be.
resource "helm_release" "fleet_crd" {
  depends_on       = [null_resource.registry_mirror]
  name             = "fleet-crd"
  repository       = "https://rancher.github.io/fleet-helm-charts/"
  chart            = "fleet-crd"
  version          = var.fleet_chart_version
  namespace        = "cattle-fleet-system"
  create_namespace = true
  wait             = true
  timeout          = 900
}

resource "helm_release" "fleet" {
  depends_on       = [helm_release.fleet_crd]
  name             = "fleet"
  repository       = "https://rancher.github.io/fleet-helm-charts/"
  chart            = "fleet"
  version          = var.fleet_chart_version
  namespace        = "cattle-fleet-system"
  create_namespace = true
  wait             = true
  timeout          = 900
}

output "cluster_name" { value = local.cluster_name }
output "cluster_kind" { value = "kind" }
output "image_registry_mirror" { value = var.image_registry_mirror }
output "kubeconfig" {
  value     = kind_cluster.this.kubeconfig
  sensitive = true
}
