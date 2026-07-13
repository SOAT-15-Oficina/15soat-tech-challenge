locals {
  kubeconfig_path = abspath("${path.module}/kubeconfig")

  common_labels = {
    "app.kubernetes.io/name" = "oficina-mecanica"
  }

  postgres_labels = merge(local.common_labels, {
    "app.kubernetes.io/component" = "database"
  })
}

# Cluster Kubernetes local provisionado com Kind. O kubeconfig e gravado em
# disco para uso por kubectl e pelo pipeline de deploy da aplicacao.
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
