# Tech Challenge - Step 1

Sistema de gerenciamento de oficina mecânica com controle de ordens de serviço, estoque e clientes.

## Banco escolhido

O projeto utiliza **PostgreSQL** como banco de dados relacional. A escolha se justifica pela natureza do domínio: uma oficina mecânica possui entidades com relacionamentos bem definidos entre clientes, veículos, ordens de serviço, serviços, peças e movimentações de estoque. O modelo relacional garante integridade referencial entre essas entidades — por exemplo, uma ordem de serviço sempre estará vinculada a um cliente e veículo válidos.

Além disso, o PostgreSQL oferece suporte robusto a transações ACID, o que é essencial para operações como baixa de estoque e mudança de status de ordens de serviço, onde consistência é crítica.

## Arquitetura

A aplicação segue uma organização inspirada em **Arquitetura Hexagonal / Clean Architecture**: o domínio, as portas de aplicação e os casos de uso ficam no centro, enquanto HTTP, PostgreSQL, e-mail, arquivos estáticos e configuração são adaptadores externos.

Regra de dependência: código de domínio e casos de uso de OS não dependem de handlers HTTP, Fiber, `pgx`, SQL, providers de e-mail, Kubernetes, Docker ou detalhes de infraestrutura. As dependências apontam para dentro: `handlers -> services/use cases -> application ports/domain`. Os adaptadores externos implementam as portas internas: `internal/repository` implementa persistência com PostgreSQL/`pgx`, e `packages/email` implementa notificações de orçamento e alertas.

Visão consolidada da solução real: API Go/Fiber, painel web estático, PostgreSQL, e-mail via MailHog no ambiente local/Kubernetes, Terraform criando um cluster Kind local, Metrics Server para HPA e GitHub Actions com runner hospedado para lint/testes e runner self-hosted para build/deploy local.

### Componentes da aplicação

```mermaid
flowchart TB
    subgraph clients [Clientes / Usuarios]
        WEB[Painel Web /web]
        SWAG[Swagger UI]
        EXT[Cliente externo]
        PUB[Cliente final - aprovacao]
    end

    subgraph inbound [Adaptadores de entrada]
        HTTP[HTTP :8080<br/>Fiber v3]
        STATIC[Static files<br/>/web]
        DOCS[Swagger<br/>/swagger e /docs/swagger.yaml]
        HEALTH[Health probes<br/>/ping e /ready]
    end

    subgraph app [Casos de uso]
        WO[Work orders<br/>abrir, listar, consultar, atualizar status]
        BUDGET[Orcamentos<br/>enviar, aprovar, recusar]
        CATALOG[Catalogos<br/>clientes, veiculos, servicos, insumos]
        AUTHZ[Autenticacao e RBAC]
    end

    subgraph api [Nucleo da aplicacao]
        H[Handlers HTTP<br/>validacao, DTOs, erros]
        S[Services<br/>WorkOrder, Creation, Status, Budget<br/>Public, Workshop, Supply, Customer...]
        APP[Application<br/>portas internas, filtros, DTOs<br/>e erros de aplicacao]
        R[Repositories pgx<br/>adaptadores de persistencia]
        DOM[Domain - entidades e enums]
        CFG[Config - env / viper]
        MIG[Database - migrations goose no boot]
    end

    subgraph outbound [Adaptadores de saida]
        PG[(PostgreSQL :5432)]
        MAILHOG[MailHog SMTP :1025<br/>UI :8025]
    end

    subgraph pkgs [Pacotes compartilhados]
        EMAILPKG[packages/email<br/>Provider + Templates HTML]
        AUTH[internal/auth<br/>JWT Claims]
    end

    WEB --> HTTP
    SWAG --> DOCS
    EXT --> HTTP
    PUB --> HTTP
    HTTP --> H
    STATIC --> H
    DOCS --> H
    HEALTH --> H

    H --> WO
    H --> BUDGET
    H --> CATALOG
    H --> AUTHZ
    WO --> S
    BUDGET --> S
    CATALOG --> S
    AUTHZ --> S
    H --> AUTH
    H --> S
    S --> APP
    S --> DOM
    R -. implementa .-> APP
    EMAILPKG -. implementa .-> APP
    EMAILPKG --> MAILHOG
    R --> DOM
    R --> PG
    MIG --> PG
    CFG -.-> H
```

| Camada | Papel |
|--------|--------|
| **Web** | Board, OS, clientes, veículos, serviços, insumos, aprovação — consome a REST API |
| **Handlers** | HTTP, validação de entrada, tradução de erros, respostas JSON |
| **Services / Use cases** | Regras de negócio: abertura de OS, máquina de estados, orçamento, criação de itens, aprovações públicas; dependem de `domain` e portas de `application` |
| **Application** | Portas internas de persistência e notificação, filtros/DTOs de listagem e erros compartilhados de aplicação |
| **Repositories** | Adaptadores SQL com `pgx`; implementam portas de persistência e traduzem erros de banco para erros de aplicação |
| **Domain** | Entidades (`WorkOrder`, `Customer`, `WorkshopService`, etc.) |
| **packages/email** | Adaptador de saída para envio de orçamento e alertas; implementa portas de notificação via provider configurável |
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

#### Ambiente Kubernetes local (Terraform + Kind + manifests em `k8s/`)

```mermaid
flowchart TB
    subgraph IaC["Infraestrutura como Codigo"]
        TF["Terraform<br/>terraform/local-kind"]
        KIND["Cluster Kind<br/>techchallenge-local"]
        KCFG["kubeconfig local"]
    end

    subgraph K8s["Cluster Kubernetes — namespace: workshop"]
        subgraph Config["Configuração"]
            CM["ConfigMap api-config<br/>env não sensível"]
            SEC["Secret api-secrets<br/>JWT · senha DB"]
            NS["Namespace workshop"]
        end

        subgraph Workloads["Workloads"]
            APID["Deployment api<br/>replicas: 2<br/>image: techchallenge/api:&lt;sha&gt;<br/>porta 8080"]
            PGD["Deployment postgres<br/>replicas: 1<br/>image: postgres:18.3<br/>porta 5432"]
            MHD["Deployment mailhog<br/>replicas: 1"]
            MS["Metrics Server<br/>kube-system"]
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

    TF --> KIND
    KIND --> KCFG
    KCFG --> K8s
    APID --> SVC_API
    PGD --> SVC_PG
    MHD --> SVC_MH
    PGD --> PVC
    APID --> CM
    APID --> SEC
    MS --> HPA
    HPA --> APID
    APID -->|"SQL"| SVC_PG
    APID -->|"SMTP"| SVC_MH
```

| Recurso | Dev (Compose) | K8s local real |
|---------|---------------|----------------|
| **API** | 1 container, build multi-stage (`Dockerfile`) | Deployment + Service `api-service` + HPA (2–10 réplicas) |
| **PostgreSQL** | 1 container custom (`database/Dockerfile`) | Deployment com imagem oficial `postgres:18.3` + Service + PVC 5Gi |
| **E-mail** | MailHog | MailHog (prod real: AWS SES — vars já no ConfigMap, provider ainda só `mailhog`) |
| **Secrets** | `.env` | `Secret` + `ConfigMap` |
| **Autoscaling** | — | Metrics Server + HPA por CPU/memória |
| **Health checks** | `/ping` e `/ready` | Liveness em `/ping`; readiness em `/ready` |
| **CI/CD** | — | GitHub Actions: lint/testes hospedados; build/deploy no runner self-hosted `local-kind` |

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

#### CI/CD (GitHub Actions — `.github/workflows/ci.yml`)

```mermaid
flowchart TD
    EVT["Pull request ou push em main"] --> LINT["lint<br/>ubuntu-latest<br/>actionlint"]
    EVT --> TEST["test<br/>ubuntu-latest<br/>go test ./..."]
    TEST --> PG17["PostgreSQL 17<br/>service container"]
    LINT --> GATE["gate"]
    TEST --> GATE

    GATE -->|"somente push em main"| SELF["runner self-hosted<br/>labels: self-hosted, local-kind"]
    GATE -->|"pull request"| ENDPR["encerra apos lint/testes"]

    SELF --> BUILD["go build ./..."]
    BUILD --> IMG["docker build<br/>techchallenge/api:&lt;sha&gt;<br/>techchallenge/api:latest"]
    IMG --> LOAD["kind load docker-image<br/>imagem local no Kind"]
    LOAD --> APPLY["kubectl apply -f k8s/"]
    APPLY --> METRICS["instala Metrics Server<br/>aguarda metrics.k8s.io"]
    METRICS --> SETIMG["kubectl set image<br/>api=techchallenge/api:&lt;sha&gt;"]
    SETIMG --> ROLLOUT["rollout api/postgres<br/>valida GET /ping"]
```

#### Deploy em Kubernetes local

```mermaid
flowchart TD
    START["Push em main"] --> CI["lint + testes OK"]
    CI --> BUILD["runner self-hosted<br/>go build + docker build API"]
    BUILD --> LOAD["kind load docker-image<br/>techchallenge/api:&lt;sha&gt;"]
    LOAD --> K0["kubectl apply -f k8s/metrics-server.yaml"]
    K0 --> K1["kubectl apply -f k8s/namespace.yaml"]
    K1 --> K2["kubectl apply -f k8s/configmap.yaml<br/>kubectl apply -f k8s/secret.yaml"]
    K2 --> K3["kubectl apply -f k8s/postgres-pvc.yaml"]
    K3 --> K4["kubectl apply -f k8s/postgres-*.yaml<br/>postgres:18.3"]
    K4 --> K5["kubectl apply -f k8s/mailhog-*.yaml"]
    K5 --> K6["kubectl apply -f k8s/api-deployment.yaml<br/>kubectl apply -f k8s/api-service.yaml"]
    K6 --> K7["kubectl apply -f k8s/api-hpa.yaml"]
    K7 --> K8["kubectl set image deployment/api<br/>techchallenge/api:&lt;sha&gt;"]

    K8 --> BOOT["Pods API sobem"]
    BOOT --> MIG["API executa migrations goose"]
    MIG --> READY["Readiness /ready → tráfego liberado"]
    READY --> SCALE["HPA escala 2–10 pods conforme CPU/memória"]
```

**Ordem de dependência no cluster**

1. Namespace `workshop`
2. ConfigMap + Secret (credenciais e env)
3. PVC (dados persistentes do Postgres)
4. Postgres (aguarda `pg_isready`)
5. MailHog (SMTP interno)
6. Metrics Server (habilita métricas para o HPA)
7. API (depende de DB + e-mail; liveness em `/ping`, readiness em `/ready`)
8. HPA (autoscaling da API)

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

### Cluster Kubernetes local com Terraform + Kind

Além do Docker Compose, o projeto possui um setup local em `terraform/local-kind` para subir um mini cluster Kubernetes com Kind.

Pré-requisitos:

- Docker
- Terraform `>= 1.6`
- Kind
- kubectl

O provider Terraform `tehcyx/kind` gerencia o cluster Kind, mas nao instala esses binarios. Eles precisam estar disponiveis no ambiente local, com o Docker em execucao. O projeto usa Kind com deploy local por maior simplicidade no fluxo de desenvolvimento e validacao.

Com `mise`, instale as ferramentas declaradas no projeto e execute o Terraform dentro do ambiente gerenciado:

```bash
mise install
mise exec -- terraform -chdir=terraform/local-kind init
mise exec -- terraform -chdir=terraform/local-kind apply
```

Sem `mise`, instale as ferramentas manualmente, confirme que estao no `PATH` e rode os comandos diretamente:

```bash
docker version
kind version
kubectl version --client
terraform version

terraform -chdir=terraform/local-kind init
terraform -chdir=terraform/local-kind apply
```

O Terraform usa o provider `tehcyx/kind` para criar somente o cluster Kind e gerar um kubeconfig local. Os manifestos em `k8s/` nao sao aplicados pelo Terraform.

Para usar `kubectl` apontando para o cluster criado:

```bash
export KUBECONFIG="$(pwd)/terraform/local-kind/kubeconfig"
kubectl get nodes
```

Para alterar o nome do cluster:

```bash
# com mise
mise exec -- terraform -chdir=terraform/local-kind apply -var='cluster_name=techchallenge-dev'

# sem mise
terraform -chdir=terraform/local-kind apply -var='cluster_name=techchallenge-dev'
```

Para remover tudo:

```bash
# com mise
mise exec -- terraform -chdir=terraform/local-kind destroy

# sem mise
terraform -chdir=terraform/local-kind destroy
```

### GitHub Actions com runner local

O workflow em `.github/workflows/ci.yml` usa dois ambientes:

- `ubuntu-latest` do GitHub para executar `actionlint` nos workflows.
- `ubuntu-latest` do GitHub para executar `go test ./...` com PostgreSQL de servico.
- Runner local com os labels `self-hosted` e `local-kind` para executar `go build ./...`, buildar a imagem Docker, carregar a imagem no Kind e aplicar os manifestos Kubernetes.

O deploy local roda somente em `push` para `main`. Pull requests executam lint e testes no runner hospedado do GitHub.

Prerequisitos na maquina do runner local:

- Docker em execucao
- Acesso ao Docker pelo usuario que executa o runner
- Go `1.26`
- Kind
- kubectl
- Terraform `>= 1.6`
- Cluster Kind criado com o nome padrao `techchallenge-local`

Instale as ferramentas declaradas no projeto com `mise`:

```bash
mise install
```

Crie ou atualize o cluster local antes de rodar o deploy:

```bash
mise exec -- terraform -chdir=terraform/local-kind init
mise exec -- terraform -chdir=terraform/local-kind apply
export KUBECONFIG="$(pwd)/terraform/local-kind/kubeconfig"
kubectl get nodes
```

Durante a execucao do workflow, o kubeconfig e regenerado com `kind get kubeconfig --name techchallenge-local` em um arquivo temporario do runner. Isso evita depender de arquivos nao versionados dentro do checkout do repositorio.

Para cadastrar o runner no GitHub:

1. Acesse o repositorio no GitHub.
2. Abra `Settings` > `Actions` > `Runners`.
3. Clique em `New self-hosted runner`.
4. Selecione `Linux` e a arquitetura da maquina.
5. Execute, na maquina local, os comandos de download e configuracao exibidos pelo GitHub.
6. Na etapa `./config.sh`, mantenha os labels padrao do GitHub e adicione o label customizado `local-kind`.
7. Instale o runner como servico:

```bash
sudo ./svc.sh install
sudo ./svc.sh start
sudo ./svc.sh status
```

Como alternativa, use o script do projeto. Ele solicita o token de registro via `gh` ou usa o token informado em `RUNNER_TOKEN`. A conta autenticada precisa ter permissao de administrador no repositorio; permissao de push nao permite registrar runners.

```bash
chmod +x scripts/setup-self-hosted-runner.sh

scripts/setup-self-hosted-runner.sh

RUNNER_TOKEN="<token-do-github>" scripts/setup-self-hosted-runner.sh
```

Para um repositorio de organizacao, informe o repositorio explicitamente. O token deve ser um token de registro temporario gerado em `Settings` > `Actions` > `Runners` > `New self-hosted runner`, e nao um Personal Access Token:

```bash
REPO=SOAT-15-Oficina/15soat-tech-challenge \
RUNNER_TOKEN="<token-de-registro>" \
scripts/setup-self-hosted-runner.sh
```

O script executa todo o fluxo: detecta a arquitetura local (`x64`, `arm64` ou `arm`), descobre a versao mais recente, baixa o pacote, valida o SHA256 publicado pelo GitHub, configura o runner com o label `local-kind`, instala-o como servico e inicia o servico. Por padrao, o servico e instalado e iniciado automaticamente.

Para apenas configurar sem instalar o servico, use `INSTALL_SERVICE=false`; para instalar mas deixar parado, use `START_SERVICE=false`. Para sobrescrever a arquitetura, informe `RUNNER_ARCH`, por exemplo:

```bash
RUNNER_ARCH=arm64 INSTALL_SERVICE=false \
RUNNER_TOKEN="<token-de-registro>" \
scripts/setup-self-hosted-runner.sh
```

Confirme que o runner aparece como `Idle` em `Settings` > `Actions` > `Runners`.

Valide os comandos que o workflow precisa executar com o mesmo usuario do runner:

```bash
docker version
kind version
kubectl --kubeconfig "$(pwd)/terraform/local-kind/kubeconfig" get nodes
go version
```

Fluxo do workflow em `push` para `main`:

1. Executa `actionlint` no runner hospedado do GitHub.
2. Executa os testes no runner hospedado do GitHub.
3. Se lint e testes passarem, executa o build no runner local.
4. Cria as tags Docker locais `techchallenge/api:<sha-do-commit>` e `techchallenge/api:latest`.
5. Carrega `techchallenge/api:<sha-do-commit>` no cluster Kind com `kind load docker-image`.
6. Aplica os manifestos em `k8s/`.
7. Instala o Metrics Server e aguarda a API `metrics.k8s.io`, usada pelo HPA.
8. Atualiza o Deployment da API para a tag imutavel do commit.
9. Garante idempotentemente os dados de demonstracao usados na consulta publica.
10. Aguarda os rollouts e valida `GET /ping` via `kubectl port-forward`.

Como a imagem e carregada diretamente no Kind, o Deployment da API usa `imagePullPolicy: IfNotPresent`. Usar `Always` faria o cluster tentar buscar a tag em um registry externo, o que nao existe neste fluxo local.

Para recuperar o cluster local apos uso com outra imagem de PostgreSQL ou outro layout de dados, recrie o PVC:

```bash
kubectl delete deployment api postgres -n workshop --ignore-not-found
kubectl delete pvc postgres-pvc -n workshop --ignore-not-found
```

Evite usar runner local self-hosted para codigo de forks ou contribuidores nao confiaveis. O runner executa comandos com acesso a Docker e ao cluster local.

### Validacao local do HPA com k6

O manifesto `k8s/metrics-server.yaml` corresponde ao manifesto oficial do
[Metrics Server v0.8.1](https://github.com/kubernetes-sigs/metrics-server/releases/tag/v0.8.1),
com `--kubelet-insecure-tls` adicionado para os certificados dos kubelets do Kind. O HPA da API usa CPU e memoria para manter entre 2 e 10 replicas.

Com o deploy aplicado ao cluster e o `KUBECONFIG` apontando para o Kind, instale as ferramentas e execute o teste demonstrativo:

```bash
mise install
mise exec -- scripts/validate-hpa.sh
```

O script valida o cluster, os pods, o Metrics Server e as metricas numericas do HPA; aguarda a linha de base de duas replicas; confirma que a OS de demonstracao criada pelo deploy responde; cria um proxy TCP temporario dentro do cluster; executa o cenario somente leitura contra `GET /public/work-orders/OS-2026-0001`; e falha se os thresholds do k6 forem violados ou se a API nao ultrapassar duas replicas disponiveis. O `port-forward` aponta para esse proxy, que entra no ClusterIP do Service e distribui as conexoes entre os pods. Isso evita tanto a selecao de um unico pod feita por `kubectl port-forward service/...` quanto colocar o API server no caminho da carga. O pod e os processos temporarios sao removidos automaticamente.

Em outro terminal, acompanhe as decisoes do HPA:

```bash
kubectl get hpa api-hpa -n workshop -w
kubectl top pods -n workshop
```

O Metrics Server publica amostras a cada 15 segundos, mas o scale-out pode levar alguns minutos entre coleta, decisao do HPA e prontidao dos novos pods. Ao fim da carga, o script observa o retorno gradual ao minimo de duas replicas. A janela padrao de estabilizacao de scale-down do Kubernetes costuma acrescentar cerca de cinco minutos; use `OBSERVE_SCALE_DOWN=false` para nao aguardar essa observacao.

Cada replica da API limita seu pool a 5 conexoes (`DATABASE_MAX_CONNECTIONS`), mantendo no maximo 50 conexoes da aplicacao quando o HPA chega a 10 pods. A liveness usa `/ping` para verificar somente o processo, enquanto a readiness usa `/ready` e retira o pod do Service quando o PostgreSQL nao responde. O PostgreSQL usa estrategia `Recreate` por compartilhar um unico PVC e sua liveness TCP evita reinicios causados apenas por saturacao transitoria.

O cenario usa aquecimento, crescimento, pico sustentado e desaceleracao. Os valores padrao podem ser sobrescritos:

```bash
VUS=300 \
WARMUP_DURATION=45s \
RAMP_DURATION=90s \
PEAK_DURATION=3m \
COOLDOWN_DURATION=45s \
P95_MS=750 \
WORK_ORDER_CODE=OS-2026-0001 \
CUSTOMER_DOCUMENT=12345678901 \
mise exec -- scripts/validate-hpa.sh
```

Tambem podem ser ajustados `WARMUP_VUS`, `LATENCY_CHECK_MS`, `REQUEST_TIMEOUT`, `SLEEP_SECONDS`, `LOCAL_PORT`, `METRICS_TIMEOUT`, `BASELINE_TIMEOUT`, `SCALE_OUT_TIMEOUT`, `SCALE_DOWN_TIMEOUT` e `LOAD_PROXY_IMAGE`. Para testar uma URL ja exposta por outro gateway ou load balancer, defina `BASE_URL`; nesse caso o proxy temporario nao e iniciado. `USE_CLUSTER_PROXY=true` permite fornecer explicitamente a URL local usada com o proxy do cluster.

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

### Notificações de status da OS

Toda transição relevante da ordem de serviço dispara um e-mail ao cliente com **código da OS**, **status anterior** e **novo status**.

| Transição | E-mail |
|-----------|--------|
| `RECEBIDA` → `EM_DIAGNOSTICO` | Atualização de status |
| `EM_DIAGNOSTICO` → `AGUARDANDO_APROVACAO` | Orçamento com links de aprovação/recusa |
| `AGUARDANDO_APROVACAO` → `APROVADO` | Atualização de status |
| `AGUARDANDO_APROVACAO` → `CANCELADA` | Atualização de status (recusa total ou cancelamento) |
| `RECEBIDA` / `EM_DIAGNOSTICO` → `CANCELADA` | Atualização de status |
| `APROVADO` → `EM_EXECUCAO` | Atualização de status |
| `EM_EXECUCAO` → `FINALIZADA` | Atualização de status |
| `FINALIZADA` → `ENTREGUE` | Atualização de status |

**Política de falha (best effort):**
- A transição de status é persistida de forma atômica e idempotente.
- Falhas no provedor de e-mail são apenas registradas em log; a API não retorna erro e não reverte o status.
- `quote_sent_at` e total do orçamento só são gravados após envio bem-sucedido do e-mail de orçamento.
- Transições repetidas para o mesmo status não geram e-mail duplicado.

O envio é feito pelo `email.Provider` de `packages/email` (MailHog em desenvolvimento), exposto à aplicação pela porta `internal/application/port.EmailSender` (alias do mesmo contrato).

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
