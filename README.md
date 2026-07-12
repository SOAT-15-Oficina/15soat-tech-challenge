# Tech Challenge - Step 1

Sistema de gerenciamento de oficina mecânica com controle de ordens de serviço, estoque e clientes.

## Banco escolhido

O projeto utiliza **PostgreSQL** como banco de dados relacional. A escolha se justifica pela natureza do domínio: uma oficina mecânica possui entidades com relacionamentos bem definidos entre clientes, veículos, ordens de serviço, serviços, peças e movimentações de estoque. O modelo relacional garante integridade referencial entre essas entidades — por exemplo, uma ordem de serviço sempre estará vinculada a um cliente e veículo válidos.

Além disso, o PostgreSQL oferece suporte robusto a transações ACID, o que é essencial para operações como baixa de estoque e mudança de status de ordens de serviço, onde consistência é crítica.

## Arquitetura

Visão consolidada da solução: API Go/Fiber, painel web estático, PostgreSQL e e-mail via MailHog (dev/K8s), com preparação para AWS SES.

### Componentes da aplicação

```mermaid
flowchart TB
    subgraph clients [Clientes / Usuarios]
        WEB[Painel Web /web]
        SWAG[Swagger UI]
        EXT[Cliente externo]
        PUB[Cliente final - aprovacao]
    end

    subgraph api [API cmd/api - Fiber v3]
        RT[Routes + Middlewares<br/>Auth JWT / RBAC / Static]
        H[Handlers HTTP<br/>Auth, Users, Customers, Vehicles<br/>Services, Supplies, Work Orders, Public]
        S[Services<br/>WorkOrder, Creation, Status, Budget<br/>Public, Workshop, Supply, Customer...]
        R[Repositories pgx<br/>WorkOrder, WorkOrderService<br/>Customer, Vehicle, User, Supply...]
        DOM[Domain - entidades e enums]
        APP[Application - DTOs de listagem]
        CFG[Config - env / viper]
        MIG[Database - migrations goose no boot]
    end

    subgraph pkgs [Pacotes compartilhados]
        EMAIL[packages/email<br/>Provider + Templates HTML]
        AUTH[internal/auth<br/>JWT Claims]
    end

    subgraph data [Persistencia]
        PG[(PostgreSQL<br/>schema + seed)]
    end

    WEB --> RT
    SWAG --> RT
    EXT --> RT
    PUB --> RT

    RT --> H
    RT --> AUTH
    H --> S
    H --> APP
    S --> R
    S --> EMAIL
    R --> DOM
    R --> PG
    MIG --> PG
    CFG -.-> RT
```

| Camada | Papel |
|--------|--------|
| **Web** | Board, OS, clientes, veículos, serviços, insumos, aprovação — consome a REST API |
| **Handlers** | HTTP, validação de entrada, tradução de erros, respostas JSON |
| **Services** | Regras de negócio: máquina de estados da OS, orçamento, criação de itens, aprovações públicas |
| **Repositories** | SQL com `pgx`; isolamento do banco |
| **Domain** | Entidades (`WorkOrder`, `Customer`, `WorkshopService`, etc.) |
| **packages/email** | Envio de orçamento (template HTML) via provider configurável |
| **Boot** | Migrations embarcadas (`go:embed`) executadas na subida da API |

### Infraestrutura provisionada

#### Ambiente local (Docker Compose)

```mermaid
flowchart LR
    subgraph Host["Máquina do desenvolvedor"]
        subgraph Compose["docker-compose.yml"]
            API["Container: api<br/>techchallenge (scratch)<br/>:8080"]
            DB["Container: db<br/>PostgreSQL 18.3<br/>schema + seed"]
            MH["Container: mailhog<br/>SMTP :1025 · UI :8025"]
        end
        DEV["Desenvolvedor<br/>localhost:8080"]
    end

    DEV --> API
    API -->|"pgx TCP"| DB
    API -->|"SMTP"| MH
```

#### Ambiente Kubernetes (manifests em `k8s/`)

```mermaid
flowchart TB
    subgraph K8s["Cluster Kubernetes — namespace: workshop"]
        subgraph Config["Configuração"]
            CM["ConfigMap api-config<br/>env não sensível"]
            SEC["Secret api-secrets<br/>JWT · senha DB"]
            NS["Namespace workshop"]
        end

        subgraph Workloads["Workloads"]
            APID["Deployment api<br/>replicas: 2<br/>image: techchallenge/api:latest"]
            PGD["Deployment postgres<br/>replicas: 1<br/>image: techchallenge/postgres:latest"]
            MHD["Deployment mailhog<br/>replicas: 1"]
        end

        subgraph Network["Rede interna"]
            SVC_API["Service api-service<br/>ClusterIP :8080"]
            SVC_PG["Service postgres-service<br/>:5432"]
            SVC_MH["Service mailhog-service<br/>:1025 / :8025"]
        end

        subgraph Storage["Armazenamento"]
            PVC["PVC postgres-pvc<br/>5Gi ReadWriteOnce"]
        end

        HPA["HPA api-hpa<br/>2–10 pods · CPU 70% · Mem 80%"]
    end

    APID --> SVC_API
    PGD --> SVC_PG
    MHD --> SVC_MH
    PGD --> PVC
    APID --> CM
    APID --> SEC
    HPA --> APID
    APID -->|"SQL"| SVC_PG
    APID -->|"SMTP"| SVC_MH
```

| Recurso | Dev (Compose) | K8s (proposto) |
|---------|---------------|----------------|
| **API** | 1 container, build multi-stage (`Dockerfile`) | Deployment + HPA (2–10 réplicas) |
| **PostgreSQL** | 1 container custom (`database/Dockerfile`) | Deployment + PVC 5Gi |
| **E-mail** | MailHog | MailHog (prod real: AWS SES — vars já no ConfigMap, provider ainda só `mailhog`) |
| **Secrets** | `.env` | `Secret` + `ConfigMap` |
| **Observabilidade** | SonarQube opcional (`docker-compose.sonar.yml`) | Health checks `/ping` (liveness/readiness) |
| **CI** | — | GitHub Actions: build + test com Postgres 17 |

> **Nota:** O repositório não inclui Ingress/LoadBalancer nem pipeline de CD. Em produção real, entraria um **Ingress** (ou Service `LoadBalancer`) na frente do `api-service` e um registry de imagens (ECR, GHCR, etc.).

### Fluxo de deploy

#### Desenvolvimento local

```mermaid
sequenceDiagram
    participant Dev as Desenvolvedor
    participant Git as Repositório Git
    participant DC as Docker Compose
    participant API as Container API
    participant DB as PostgreSQL
    participant MH as MailHog

    Dev->>Git: git clone + cp .env.example .env
    Dev->>DC: docker compose up --build -d
    DC->>DB: sobe Postgres (schema init + seed)
    DC->>MH: sobe MailHog
    DC->>API: build Go → imagem scratch
    API->>DB: conecta + goose migrations (boot)
    API->>MH: configura provider de e-mail
    Dev->>API: http://localhost:8080/web
```

#### CI (GitHub Actions — `.github/workflows/ci.yml`)

```mermaid
flowchart LR
    A["Push / PR → main"] --> B["checkout"]
    B --> C["setup-go 1.26"]
    C --> D["go build ./..."]
    D --> E["go test ./..."]
    E --> F["Postgres 17<br/>(service container)"]
    F --> E
    E --> G["✓ ou ✗ pipeline"]
```

#### Deploy em Kubernetes (fluxo proposto)

```mermaid
flowchart TD
    START["Código mergeado em main"] --> CI["CI: build + test OK"]
    CI --> IMG1["docker build -t techchallenge/api:latest"]
    CI --> IMG2["docker build -t techchallenge/postgres:latest<br/>(context: database/)"]
    IMG1 --> PUSH["Push imagens → registry"]
    IMG2 --> PUSH

    PUSH --> K1["kubectl apply -f k8s/namespace.yaml"]
    K1 --> K2["kubectl apply -f k8s/configmap.yaml<br/>kubectl apply -f k8s/secret.yaml"]
    K2 --> K3["kubectl apply -f k8s/postgres-pvc.yaml"]
    K3 --> K4["kubectl apply -f k8s/postgres-*.yaml"]
    K4 --> K5["kubectl apply -f k8s/mailhog-*.yaml"]
    K5 --> K6["kubectl apply -f k8s/api-deployment.yaml<br/>kubectl apply -f k8s/api-service.yaml"]
    K6 --> K7["kubectl apply -f k8s/api-hpa.yaml"]

    K6 --> BOOT["Pods API sobem"]
    BOOT --> MIG["API executa migrations goose"]
    MIG --> READY["Readiness /ping → tráfego liberado"]
    READY --> SCALE["HPA escala 2–10 pods conforme CPU/memória"]
```

**Ordem de dependência no cluster**

1. Namespace `workshop`
2. ConfigMap + Secret (credenciais e env)
3. PVC (dados persistentes do Postgres)
4. Postgres (aguarda `pg_isready`)
5. MailHog (SMTP interno)
6. API (depende de DB + e-mail; probes em `/ping`)
7. HPA (autoscaling da API)

### Visão de contexto

```mermaid
flowchart LR
    A["Atendente / Admin"] -->|"HTTPS"| S["Sistema Oficina Mecânica<br/>OS, estoque, clientes, orçamentos"]
    C["Cliente"] -->|"Links públicos de aprovação"| S
    S -->|"Envia orçamentos"| E["Serviço de E-mail<br/>MailHog (dev) / SES (futuro)"]
```

## Setup

### Pré-requisitos

- Docker e Docker Compose

### Subindo o projeto

```bash
cp .env.example .env
docker compose up --build -d
```

A API fica disponível em `http://localhost:8080`.

O painel web fica acessível em `http://localhost:8080/web` (a rota `/` redireciona automaticamente para `/web`).

O banco de dados PostgreSQL é inicializado automaticamente com o schema (via `initdb.d` e goose no boot da API) e dados de seed na primeira execução. As migrations são embedded no binário da API via `go:embed`.

### SonarQube local

O SonarQube roda em um compose separado para não alterar o fluxo padrão da aplicação:

```bash
docker compose -f docker-compose.sonar.yml up -d sonarqube
```

Acesse `http://localhost:9000`, entre com `admin` / `admin` e troque a senha para `password`. Em seguida, crie um projeto local com a chave `15soat-tech-challenge-step-1` e gere um token com o valor `15soat-tech-challenge-step-1`.

> As credenciais já estão configuradas em `sonar-project.properties` (`sonar.login`, `sonar.password` e `sonar.token`), então nenhuma variável de ambiente adicional é necessária.

Antes da análise, gere o relatório de cobertura Go:

```bash
mise exec -- go test ./... -coverprofile=coverage.out
```

Se não estiver usando `mise`, rode o mesmo comando com o `go` instalado localmente:

```bash
go test ./... -coverprofile=coverage.out
```

Execute a análise:

```bash
docker compose -f docker-compose.sonar.yml run --rm sonar-scanner
```

A configuração está em `sonar-project.properties`. A análise considera `cmd`, `database`, `internal` e `packages`, e exclui explicitamente a interface web em `web/**`.

### Painel Web

O projeto inclui uma interface web para gerenciamento da oficina, acessível em [http://localhost:8080/web](http://localhost:8080/web). A rota raiz (`/`) redireciona automaticamente para o painel.

O painel permite gerenciar:

- **Board** — visão geral das ordens de serviço por status
- **Ordens de Serviço** — listagem e criação de novas OS
- **Clientes** — cadastro e consulta de clientes
- **Veículos** — cadastro e consulta de veículos
- **Serviços** — catálogo de serviços da oficina
- **Insumos** — controle de peças e insumos em estoque
- **Aprovação** — aprovação/rejeição de orçamentos pelo cliente

O login é feito com as credenciais cadastradas via `/auth/register`. Os arquivos ficam em `web/` e são servidos como conteúdo estático pelo Fiber.

### Documentacao da API (Swagger)

Com o projeto rodando, acesse o Swagger UI para visualizar e testar todos os endpoints:

- **Swagger UI**: [http://localhost:8080/swagger](http://localhost:8080/swagger)
- **OpenAPI spec**: [http://localhost:8080/docs/swagger.yaml](http://localhost:8080/docs/swagger.yaml)

Para endpoints autenticados, clique em **Authorize** no Swagger UI e insira o token JWT de desenvolvimento (seção abaixo).

### Endpoints

| Metodo | Rota    | Descricao    |
| ------ | ------- | ------------ |
| GET    | `/`        | Redireciona para `/web` |
| GET    | `/ping`    | Health check |
| GET    | `/web`     | Painel web da oficina |
| GET    | `/swagger` | Swagger UI |

## Estrutura do projeto

```
cmd/
  api/              # Entrypoint da API
web/                # Painel web (HTML/JS/CSS servido como estático)
database/
  migrations/       # Arquivos de migration do goose (embedded no binário)
  seed-files/       # Dados de seed (desenvolvimento)
  database.go       # Exporta as migrations via embed.FS
  Dockerfile        # Imagem do PostgreSQL com schema e seed
internal/
  config/           # Configuração via variáveis de ambiente
  database/         # Execução de migrations no boot da API
```

## Banco de dados

### Diagrama de tabelas

```
users ──────────────┐
                    │
customers           │
  └── vehicles      │
                    │
services            │
                    │
supplies            │
                    │
work_orders ────────┘
  └── work_order_services
        └── work_order_service_supplies
```

### Tabelas

| Tabela                         | Função                                                                                                                                                    |
| ------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `users`                        | Usuários administrativos do sistema (atendente, mecânico, administrador, controlador de estoque). Base para autenticação JWT e rastreabilidade das ações. |
| `customers`                    | Dados dos clientes da oficina (nome, documento, contatos). Usada no cadastro e identificação do cliente para abertura da ordem de serviço.                |
| `vehicles`                     | Veículos vinculados aos clientes (placa, marca, modelo, ano). Permite associar um veículo a uma ordem de serviço.                                         |
| `services`                     | Catálogo de serviços oferecidos pela oficina (troca de óleo, alinhamento, etc). Guarda preço base e tempo estimado de execução.                           |
| `supplies`                     | Peças e insumos cadastrados no sistema. Controla quantidade em estoque e disponibilidade para execução dos serviços.                                      |
| `work_orders`                  | Ordem de serviço principal. Centraliza o atendimento ligando cliente, veículo, status, técnico responsável e dados do orçamento/execução.                 |
| `work_order_services`          | Cada serviço incluído em uma ordem de serviço. Controla aprovação, execução, tempo e preço no contexto daquela OS.                                        |
| `work_order_service_supplies`  | Peças/insumos previstos ou utilizados em cada serviço de uma OS. Snapshot operacional com quantidade e preço no momento da inclusão.                       |

## Convenções da API

### Rotas

- Rotas usam **plural em inglês** e **kebab-case** para caminhos compostos: `/services`, `/services/avg-execution-time`
- Padrão REST para CRUD: `POST /recurso`, `GET /recurso`, `GET /recurso/:id`, `PUT /recurso/:id`, `DELETE /recurso/:id`
- Rotas de consulta/relatório ficam como sub-rotas do recurso: `GET /services/avg-execution-time`

### JSON (request e response)

- Campos usam **camelCase**: `estimatedTimeMinutes`, `createdAt`, `serviceId`
- Campos internos do domínio (domain structs) usam **snake_case** nas tags JSON (`price_cents`, `estimated_time_minutes`) — esses não são expostos diretamente na API
- O handler é responsável por converter entre o formato interno e o formato da API (ex: `price_cents` → `price` em reais)

### Query parameters

- Query params usam **camelCase**: `?active=true&title=oil&technicianId=uuid`
- Paginação: `page` e `limit` (padrão: page=1, limit=10)
- Filtros booleanos aceitam `true` ou `false`
- Datas usam formato `YYYY-MM-DD`: `?from=2026-01-01&to=2026-12-31`

### Mensagens de erro

- Erros são retornados em **inglês** no formato `{"error": "mensagem"}`
- Exemplos: `"service not found"`, `"invalid id"`, `"title, price and estimatedTimeMinutes are required"`
- Erros de domínio são propagados como texto: `"title is required"`, `"price must be greater than zero"`

### Códigos HTTP

| Código | Uso |
| ------ | --- |
| 200    | Sucesso em leitura ou atualização |
| 201    | Recurso criado com sucesso |
| 204    | Recurso deletado com sucesso (sem body) |
| 400    | Dados inválidos ou campos obrigatórios faltando |
| 404    | Recurso não encontrado |
| 409    | Conflito (ex: título duplicado) |
| 500    | Erro interno |

## JWT para desenvolvimento

Token sem expiração, gerado com o `JWT_SECRET_KEY` padrão do `.env.example` (`jwt-token`), role `admin`:

```
eyJhbGciOiAiSFMyNTYiLCAidHlwIjogIkpXVCJ9.eyJyb2xlIjogImFkbWluIiwgInVzZXIiOiAiZGV2In0.sxWQewGk1XDLzwM4TYXRok7MhtgTy79qEs_nMk5FOr4
```

Uso:

```bash
curl -H "Authorization: Bearer eyJhbGciOiAiSFMyNTYiLCAidHlwIjogIkpXVCJ9.eyJyb2xlIjogImFkbWluIiwgInVzZXIiOiAiZGV2In0.sxWQewGk1XDLzwM4TYXRok7MhtgTy79qEs_nMk5FOr4" \
  http://localhost:8080/work-orders
```

## Variáveis de ambiente

| Variável             | Descrição              | Valor padrão       |
| -------------------- | ---------------------- | ------------------ |
| `SERVER_PORT`        | Porta da API           | `8080`             |
| `SERVER_ENVIRONMENT` | Ambiente de execução   | `development`      |
| `DATABASE_USER`      | Usuário do PostgreSQL  | `techchallenge`    |
| `DATABASE_PASSWORD`  | Senha do PostgreSQL    | `password`         |
| `DATABASE_HOST`      | Host do PostgreSQL     | `db`               |
| `DATABASE_PORT`      | Porta do PostgreSQL    | `5432`             |
| `DATABASE_NAME`      | Nome do banco          | `techchallenge-db` |
| `JWT_SECRET_KEY`     | Chave secreta para JWT | `jwt-token`        |
