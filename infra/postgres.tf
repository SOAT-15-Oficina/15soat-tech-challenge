locals {
  postgres_service_name = "postgres-service"
}

# Namespace da aplicacao. A camada de app (API, MailHog) e implantada aqui pelo
# pipeline, mas o namespace e a base compartilhada gerenciada pelo Terraform.
resource "kubernetes_namespace" "workshop" {
  metadata {
    name   = var.namespace
    labels = local.common_labels
  }
}

# Configuracao nao sensivel compartilhada por PostgreSQL e API. As chaves de
# banco derivam das variaveis para manter uma unica fonte de verdade.
resource "kubernetes_config_map" "api_config" {
  metadata {
    name      = "api-config"
    namespace = kubernetes_namespace.workshop.metadata[0].name
    labels    = local.common_labels
  }

  data = {
    SERVER_ENVIRONMENT       = "production"
    SERVER_PORT              = "8080"
    APP_BASE_URL             = "http://localhost:8080"
    DATABASE_HOST            = local.postgres_service_name
    DATABASE_PORT            = "5432"
    DATABASE_NAME            = var.database_name
    DATABASE_USER            = var.database_user
    DATABASE_MAX_CONNECTIONS = "5"
    EMAIL_PROVIDER           = "mailhog"
    EMAIL_HOST               = "mailhog-service"
    EMAIL_PORT               = "1025"
    EMAIL_FROM               = "oficina@workshop.local"
    AWS_DEFAULT_REGION       = "sa-east-1"
    SES_SENDER_EMAIL         = "foo@bar.com"
    SES_REPLY_TO             = "foo@bar.com"
  }
}

# Segredos compartilhados por PostgreSQL e API.
resource "kubernetes_secret" "api_secrets" {
  metadata {
    name      = "api-secrets"
    namespace = kubernetes_namespace.workshop.metadata[0].name
    labels    = local.common_labels
  }

  type = "Opaque"

  data = {
    DATABASE_PASSWORD = var.database_password
    JWT_SECRET_KEY    = var.jwt_secret_key
  }
}

# Persistencia do PostgreSQL.
resource "kubernetes_persistent_volume_claim" "postgres" {
  metadata {
    name      = "postgres-pvc"
    namespace = kubernetes_namespace.workshop.metadata[0].name
    labels    = local.postgres_labels
  }

  spec {
    access_modes = ["ReadWriteOnce"]

    resources {
      requests = {
        storage = var.postgres_storage
      }
    }
  }

  # Kind provisiona o volume por demanda quando o pod monta o PVC; sem esse
  # wait, o `apply` bloquearia esperando um bind que so ocorre no primeiro uso.
  wait_until_bound = false
}

# Deployment do PostgreSQL.
resource "kubernetes_deployment" "postgres" {
  metadata {
    name      = "postgres"
    namespace = kubernetes_namespace.workshop.metadata[0].name
    labels    = local.postgres_labels
  }

  spec {
    replicas = 1

    strategy {
      type = "Recreate"
    }

    selector {
      match_labels = local.postgres_labels
    }

    template {
      metadata {
        labels = local.postgres_labels
      }

      spec {
        container {
          name              = "postgres"
          image             = var.postgres_image
          image_pull_policy = "IfNotPresent"

          port {
            container_port = 5432
            name           = "postgres"
          }

          env {
            name = "POSTGRES_USER"
            value_from {
              config_map_key_ref {
                name = kubernetes_config_map.api_config.metadata[0].name
                key  = "DATABASE_USER"
              }
            }
          }

          env {
            name = "POSTGRES_PASSWORD"
            value_from {
              secret_key_ref {
                name = kubernetes_secret.api_secrets.metadata[0].name
                key  = "DATABASE_PASSWORD"
              }
            }
          }

          env {
            name = "POSTGRES_DB"
            value_from {
              config_map_key_ref {
                name = kubernetes_config_map.api_config.metadata[0].name
                key  = "DATABASE_NAME"
              }
            }
          }

          resources {
            requests = {
              cpu    = "250m"
              memory = "256Mi"
            }
            limits = {
              cpu    = "1"
              memory = "512Mi"
            }
          }

          volume_mount {
            name       = "postgres-data"
            mount_path = "/var/lib/postgresql"
          }

          liveness_probe {
            tcp_socket {
              port = "postgres"
            }
            initial_delay_seconds = 10
            period_seconds        = 10
            timeout_seconds       = 3
            failure_threshold     = 6
          }

          readiness_probe {
            exec {
              command = ["pg_isready", "-U", var.database_user, "-d", var.database_name]
            }
            initial_delay_seconds = 5
            period_seconds        = 5
            timeout_seconds       = 3
          }
        }

        volume {
          name = "postgres-data"
          persistent_volume_claim {
            claim_name = kubernetes_persistent_volume_claim.postgres.metadata[0].name
          }
        }
      }
    }
  }

  # O deployment cria o pod que monta o PVC; nao aguardar o rollout evita
  # travar o apply enquanto a imagem e baixada no primeiro provisionamento.
  wait_for_rollout = false
}

# Service ClusterIP que expoe o PostgreSQL para a aplicacao.
resource "kubernetes_service" "postgres" {
  metadata {
    name      = local.postgres_service_name
    namespace = kubernetes_namespace.workshop.metadata[0].name
    labels    = local.postgres_labels
  }

  spec {
    type     = "ClusterIP"
    selector = local.postgres_labels

    port {
      name        = "postgres"
      port        = 5432
      target_port = "postgres"
    }
  }
}
