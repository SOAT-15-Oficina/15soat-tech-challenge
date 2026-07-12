variable "cluster_name" {
  description = "Nome do cluster Kind local."
  type        = string
  default     = "techchallenge-local"
}

variable "node_image" {
  description = "Imagem kindest/node usada pelo Kind. Null usa o default do provider."
  type        = string
  default     = null
}

variable "namespace" {
  description = "Namespace onde os workloads (PostgreSQL e MailHog) sao aplicados."
  type        = string
  default     = "workshop"
}

variable "apply_workloads" {
  description = "Aplica os manifestos K8s (PostgreSQL e MailHog) no cluster apos cria-lo."
  type        = bool
  default     = true
}
