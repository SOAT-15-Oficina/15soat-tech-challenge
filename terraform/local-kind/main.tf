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
  repo_root       = abspath("${path.module}/../..")
  kubeconfig_path = abspath("${path.module}/kubeconfig")
  manifests_path  = abspath("${local.repo_root}/k8s")
  api_image       = "techchallenge/api:latest"
  postgres_image  = "techchallenge/postgres:latest"
  api_source_files = concat(
    [
      "Dockerfile",
      "go.mod",
      "go.sum",
      "docs/swagger.yaml",
    ],
    tolist(fileset(local.repo_root, "cmd/**/*.go")),
    tolist(fileset(local.repo_root, "internal/**/*.go")),
    tolist(fileset(local.repo_root, "packages/**/*.go")),
    tolist(fileset(local.repo_root, "web/**/*"))
  )
  postgres_source_files = concat(
    [
      "database/Dockerfile",
    ],
    tolist(fileset(local.repo_root, "database/migrations/*.sql")),
    tolist(fileset(local.repo_root, "database/seed-files/*.sql"))
  )
  api_source_hash = sha256(join("", [
    for f in local.api_source_files :
    filesha256("${local.repo_root}/${f}")
  ]))
  postgres_source_hash = sha256(join("", [
    for f in local.postgres_source_files :
    filesha256("${local.repo_root}/${f}")
  ]))
  manifests_hash = sha256(join("", [
    for f in fileset(local.manifests_path, "**/*.yaml") :
    filesha256("${local.manifests_path}/${f}")
  ]))
}

resource "kind_cluster" "local" {
  name            = var.cluster_name
  kubeconfig_path = local.kubeconfig_path
  node_image      = var.node_image
  wait_for_ready  = true

  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"

    node {
      role = "control-plane"
    }
  }
}

resource "terraform_data" "images" {
  count = var.apply_workloads ? 1 : 0

  depends_on = [kind_cluster.local]

  triggers_replace = {
    api_source_hash      = local.api_source_hash
    postgres_source_hash = local.postgres_source_hash
  }

  input = {
    api_image      = local.api_image
    postgres_image = local.postgres_image
    cluster_name   = kind_cluster.local.name
    repo_root      = local.repo_root
  }

  provisioner "local-exec" {
    command = <<-EOT
      set -eu

      for bin in docker kind; do
        if ! command -v "$bin" >/dev/null 2>&1; then
          echo "Missing required command: $bin" >&2
          exit 1
        fi
      done

      docker build -t "${self.input.api_image}" "${self.input.repo_root}"
      docker build -t "${self.input.postgres_image}" "${self.input.repo_root}/database"

      kind load docker-image "${self.input.api_image}" --name "${self.input.cluster_name}"
      kind load docker-image "${self.input.postgres_image}" --name "${self.input.cluster_name}"
    EOT
  }
}

resource "terraform_data" "workloads" {
  count = var.apply_workloads ? 1 : 0

  depends_on = [terraform_data.images[0]]

  triggers_replace = {
    cluster_id     = kind_cluster.local.id
    images_id      = terraform_data.images[0].id
    manifests_hash = local.manifests_hash
  }

  input = {
    kubeconfig     = local.kubeconfig_path
    manifests_path = local.manifests_path
    namespace      = var.namespace
  }

  provisioner "local-exec" {
    command = <<-EOT
      set -eu
      export KUBECONFIG="${local.kubeconfig_path}"

      if ! command -v kubectl >/dev/null 2>&1; then
        echo "Missing required command: kubectl" >&2
        exit 1
      fi

      kubectl apply -f "${local.manifests_path}/namespace.yaml"
      kubectl apply -R -f "${local.manifests_path}"
      kubectl -n "${var.namespace}" rollout restart deployment/postgres deployment/api

      kubectl -n "${var.namespace}" rollout status deployment/postgres --timeout=180s
      kubectl -n "${var.namespace}" rollout status deployment/mailhog --timeout=180s
      kubectl -n "${var.namespace}" rollout status deployment/api --timeout=180s
    EOT
  }

  provisioner "local-exec" {
    when       = destroy
    on_failure = continue
    command    = <<-EOT
      export KUBECONFIG="${self.input.kubeconfig}"
      kubectl delete -R -f "${self.input.manifests_path}" --ignore-not-found=true || true
    EOT
  }
}

output "kubeconfig" {
  value = kind_cluster.local.kubeconfig_path
}

output "namespace" {
  value = var.namespace
}
