terraform {
  required_version = ">= 1.6.0"

  required_providers {
    kind = {
      source  = "tehcyx/kind"
      version = "0.11.0"
    }
  }
}

provider "kind" {}

locals {
  kubeconfig_path = abspath("${path.module}/kubeconfig")
}

resource "kind_cluster" "local" {
  name            = var.cluster_name
  kubeconfig_path = local.kubeconfig_path
  wait_for_ready  = true

  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"

    node {
      role = "control-plane"
    }
  }
}

output "kubeconfig" {
  value = kind_cluster.local.kubeconfig_path
}
