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

### Endpoints

| Método | Rota    | Descrição    |
| ------ | ------- | ------------ |
| GET    | `/ping` | Health check |

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
  └── service_items │
        └── items   │
                    │
work_orders ────────┤
  ├── work_order_services
  │     └── work_order_service_items
  ├── work_order_status_history
  └── inventory_movements
```

### Tabelas

| Tabela                      | Função                                                                                                                                                    |
| --------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `users`                     | Usuários administrativos do sistema (atendente, mecânico, administrador, controlador de estoque). Base para autenticação JWT e rastreabilidade das ações. |
| `customers`                 | Dados dos clientes da oficina (nome, documento, contatos). Usada no cadastro e identificação do cliente para abertura da ordem de serviço.                |
| `vehicles`                  | Veículos vinculados aos clientes (placa, marca, modelo, ano). Permite associar um veículo a uma ordem de serviço.                                         |
| `services`                  | Catálogo de serviços oferecidos pela oficina (troca de óleo, alinhamento, etc). Guarda preço base e tempo estimado de execução.                           |
| `items`                     | Peças e insumos cadastrados no sistema. Controla quantidade em estoque e disponibilidade para execução dos serviços.                                      |
| `service_items`             | Composição padrão de um serviço: quais peças/insumos são normalmente necessários e em qual quantidade.                                                    |
| `work_orders`               | Ordem de serviço principal. Centraliza o atendimento ligando cliente, veículo, status, técnico responsável e dados do orçamento/execução.                 |
| `work_order_services`       | Cada serviço incluído em uma ordem de serviço. Controla aprovação, execução, técnico responsável, tempo e preço no contexto daquela OS.                   |
| `work_order_service_items`  | Itens previstos ou utilizados em cada serviço de uma OS. Snapshot operacional, separado da composição padrão do catálogo.                                 |
| `inventory_movements`       | Movimentações de estoque (entrada, saída, ajuste). Mantém histórico para auditoria e rastreio de baixas de peças e insumos.                               |
| `work_order_status_history` | Histórico de mudança de status da OS, permitindo rastrear a evolução (recebida, em diagnóstico, em execução, entregue, etc).                              |

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
