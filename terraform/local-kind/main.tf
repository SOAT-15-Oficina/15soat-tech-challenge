terraform {
  required_version = ">= 1.6.0"
}

locals {
  kubeconfig_path  = abspath("${path.module}/.generated/kubeconfig")
  kind_config_path = abspath("${path.module}/kind-config.yaml.tftpl")
  rendered_kind_config = templatefile(local.kind_config_path, {
    cluster_name = var.cluster_name
  })
}

resource "terraform_data" "kind_cluster" {
  input = {
    cluster_name = var.cluster_name
    kind_config  = sha256(local.rendered_kind_config)
  }

  provisioner "local-exec" {
    command = <<-EOT
      set -eu
      mkdir -p "$(dirname "${local.kubeconfig_path}")"

      for bin in docker kind kubectl; do
        if ! command -v "$bin" >/dev/null 2>&1; then
          echo "Missing required command: $bin" >&2
          exit 1
        fi
      done

      if kind get clusters | grep -qx "${var.cluster_name}"; then
        echo "Kind cluster ${var.cluster_name} already exists."
      else
        cat > "${path.module}/.generated/kind-config.yaml" <<'YAML'
${local.rendered_kind_config}
YAML
        kind create cluster \
          --name "${var.cluster_name}" \
          --config "${path.module}/.generated/kind-config.yaml" \
          --kubeconfig "${local.kubeconfig_path}"
      fi

      kind export kubeconfig --name "${var.cluster_name}" --kubeconfig "${local.kubeconfig_path}"
      kubectl --kubeconfig "${local.kubeconfig_path}" cluster-info
    EOT
  }

  provisioner "local-exec" {
    when    = destroy
    command = "kind delete cluster --name '${self.input.cluster_name}' || true"
  }
}

output "kubeconfig" {
  value = local.kubeconfig_path
}
