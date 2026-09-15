# k8s-cluster on target aws: EKS in the network's public subnets, a managed
# node group, API access limited to the machine running the IDP worker, and
# Argo CD installed.
terraform {
  required_providers {
    aws  = { source = "hashicorp/aws", version = "~> 6.0" }
    helm = { source = "hashicorp/helm", version = "~> 3.3" }
    http = { source = "hashicorp/http", version = "~> 3.5" }
  }
}

variable "name" { type = string }
variable "region" { type = string }
variable "kubernetes_version" { type = string }
variable "node_instance_type" { type = string }
variable "node_count" { type = number }
variable "argocd_chart_version" { type = string }
variable "image_registry_mirror" { type = string }
variable "tags" { type = map(string) }
variable "network" {
  description = "Outputs of the required network resource"
  type        = any
}

provider "aws" {
  region = var.region
  default_tags { tags = var.tags }
}

data "http" "worker_ip" {
  url = "https://checkip.amazonaws.com"
}

locals {
  subnets   = jsondecode(var.network.public_subnet_ids)
  allowlist = "${chomp(data.http.worker_ip.response_body)}/32"
}

data "aws_iam_policy_document" "cluster_assume" {
  statement {
    actions = ["sts:AssumeRole", "sts:TagSession"]
    principals {
      type        = "Service"
      identifiers = ["eks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "cluster" {
  name               = "${var.name}-cluster"
  assume_role_policy = data.aws_iam_policy_document.cluster_assume.json
}

resource "aws_iam_role_policy_attachment" "cluster" {
  role       = aws_iam_role.cluster.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
}

resource "aws_eks_cluster" "this" {
  name     = var.name
  version  = var.kubernetes_version
  role_arn = aws_iam_role.cluster.arn

  access_config {
    authentication_mode                         = "API"
    bootstrap_cluster_creator_admin_permissions = true
  }
  vpc_config {
    subnet_ids              = local.subnets
    endpoint_public_access  = true
    endpoint_private_access = true
    public_access_cidrs     = [local.allowlist]
  }
  depends_on = [aws_iam_role_policy_attachment.cluster]
}

data "aws_iam_policy_document" "node_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "node" {
  name               = "${var.name}-node"
  assume_role_policy = data.aws_iam_policy_document.node_assume.json
}

resource "aws_iam_role_policy_attachment" "node" {
  for_each = toset([
    "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
    "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
    "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
  ])
  role       = aws_iam_role.node.name
  policy_arn = each.value
}

resource "aws_eks_node_group" "this" {
  cluster_name    = aws_eks_cluster.this.name
  node_group_name = "${var.name}-nodes"
  node_role_arn   = aws_iam_role.node.arn
  subnet_ids      = local.subnets
  instance_types  = [var.node_instance_type]
  ami_type        = "AL2023_x86_64_STANDARD"
  scaling_config {
    desired_size = var.node_count
    min_size     = var.node_count
    max_size     = var.node_count
  }
  depends_on = [aws_iam_role_policy_attachment.node]
}

provider "helm" {
  kubernetes = {
    host                   = aws_eks_cluster.this.endpoint
    cluster_ca_certificate = base64decode(aws_eks_cluster.this.certificate_authority[0].data)
    exec = {
      api_version = "client.authentication.k8s.io/v1beta1"
      command     = "aws"
      args        = ["eks", "get-token", "--cluster-name", aws_eks_cluster.this.name, "--region", var.region]
    }
  }
}

resource "helm_release" "argocd" {
  depends_on       = [aws_eks_node_group.this]
  name             = "argocd"
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  version          = var.argocd_chart_version
  namespace        = "argocd"
  create_namespace = true
  wait             = true
  timeout          = 900
  values = [yamlencode({
    dex            = { enabled = false }
    notifications  = { enabled = false }
    applicationSet = { replicas = 0 }
    configs = {
      cm     = { "timeout.reconciliation" = "20s" }
      params = { "server.insecure" = true }
    }
  })]
}

output "cluster_name" { value = aws_eks_cluster.this.name }
output "cluster_kind" { value = "eks" }
output "endpoint" { value = aws_eks_cluster.this.endpoint }
output "ca_data" { value = aws_eks_cluster.this.certificate_authority[0].data }
output "region" { value = var.region }
output "image_registry_mirror" { value = var.image_registry_mirror }
