variable "cluster_name" {
  description = "Nome do cluster Kind local."
  type        = string
  default     = "techchallenge-local"
}

variable "namespace" {
  description = "Namespace Kubernetes onde os recursos da aplicacao sao criados."
  type        = string
  default     = "workshop"
}

variable "postgres_image" {
  description = "Imagem do container PostgreSQL."
  type        = string
  default     = "postgres:18.3"
}

variable "postgres_storage" {
  description = "Tamanho do volume persistente do PostgreSQL."
  type        = string
  default     = "5Gi"
}

variable "database_name" {
  description = "Nome do banco de dados criado no PostgreSQL."
  type        = string
  default     = "techchallenge-db"
}

variable "database_user" {
  description = "Usuario do banco de dados PostgreSQL."
  type        = string
  default     = "techchallenge"
}

variable "database_password" {
  description = "Senha do banco de dados PostgreSQL (ambiente local)."
  type        = string
  default     = "password"
  sensitive   = true
}

variable "jwt_secret_key" {
  description = "Chave usada pela API para assinar tokens JWT (ambiente local)."
  type        = string
  default     = "jwt-token"
  sensitive   = true
}
