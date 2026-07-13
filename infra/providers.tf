# O provider Kind gerencia o cluster Kubernetes local; ele nao instala os
# binarios do Kind/Docker, que precisam existir no ambiente.
provider "kind" {}

# O provider Kubernetes e configurado a partir dos atributos expostos pelo
# cluster Kind recem-criado. Assim um unico `terraform apply` sobe o cluster e,
# em seguida, os recursos do PostgreSQL, sem depender de um kubeconfig em disco.
provider "kubernetes" {
  host                   = kind_cluster.local.endpoint
  client_certificate     = kind_cluster.local.client_certificate
  client_key             = kind_cluster.local.client_key
  cluster_ca_certificate = kind_cluster.local.cluster_ca_certificate
}
