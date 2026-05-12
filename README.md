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

### Documentacao da API (Swagger)

Com o projeto rodando, acesse o Swagger UI para visualizar e testar todos os endpoints:

- **Swagger UI**: [http://localhost:8080/swagger](http://localhost:8080/swagger)
- **OpenAPI spec**: [http://localhost:8080/docs/swagger.yaml](http://localhost:8080/docs/swagger.yaml)

Para endpoints autenticados, clique em **Authorize** no Swagger UI e insira o token JWT de desenvolvimento (seção abaixo).

### Endpoints

| Metodo | Rota    | Descricao    |
| ------ | ------- | ------------ |
| GET    | `/ping` | Health check |
| GET    | `/swagger` | Swagger UI |

## Estrutura do projeto

```
cmd/
  api/              # Entrypoint da API
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
