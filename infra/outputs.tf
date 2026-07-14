output "cluster_name" {
  description = "Nome do cluster Kind provisionado."
  value       = kind_cluster.local.name
}

output "kubeconfig_path" {
  description = "Caminho do kubeconfig gerado para acessar o cluster."
  value       = kind_cluster.local.kubeconfig_path
}

output "cluster_endpoint" {
  description = "Endpoint da API do cluster Kubernetes."
  value       = kind_cluster.local.endpoint
}

output "namespace" {
  description = "Namespace onde os recursos da aplicacao sao criados."
  value       = kubernetes_namespace.workshop.metadata[0].name
}

output "postgres_service_host" {
  description = "Host DNS interno do Service do PostgreSQL."
  value       = "${kubernetes_service.postgres.metadata[0].name}.${kubernetes_namespace.workshop.metadata[0].name}.svc.cluster.local"
}

output "postgres_service_port" {
  description = "Porta do Service do PostgreSQL."
  value       = kubernetes_service.postgres.spec[0].port[0].port
}

output "database_name" {
  description = "Nome do banco de dados PostgreSQL."
  value       = var.database_name
}

output "kubectl_env" {
  description = "Comando para apontar o kubectl ao cluster provisionado."
  value       = "export KUBECONFIG=${kind_cluster.local.kubeconfig_path}"
}
