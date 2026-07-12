# Tech Challenge - Step 1

Sistema de gerenciamento de oficina mecânica com controle de ordens de serviço, estoque e clientes.

## Banco escolhido

O projeto utiliza **PostgreSQL** como banco de dados relacional. A escolha se justifica pela natureza do domínio: uma oficina mecânica possui entidades com relacionamentos bem definidos entre clientes, veículos, ordens de serviço, serviços, peças e movimentações de estoque. O modelo relacional garante integridade referencial entre essas entidades — por exemplo, uma ordem de serviço sempre estará vinculada a um cliente e veículo válidos.

Além disso, o PostgreSQL oferece suporte robusto a transações ACID, o que é essencial para operações como baixa de estoque e mudança de status de ordens de serviço, onde consistência é crítica.

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

Como alternativa, use o script do projeto. Ele solicita o token de registro via `gh` ou usa o token informado em `RUNNER_TOKEN`.

```bash
chmod +x scripts/setup-self-hosted-runner.sh

scripts/setup-self-hosted-runner.sh

RUNNER_TOKEN="<token-do-github>" scripts/setup-self-hosted-runner.sh
```

O script detecta a arquitetura local automaticamente (`x64`, `arm64` ou `arm`). Para sobrescrever a deteccao, informe `RUNNER_ARCH`, por exemplo:

```bash
RUNNER_ARCH=arm64 RUNNER_TOKEN="<token-do-github>" scripts/setup-self-hosted-runner.sh
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
