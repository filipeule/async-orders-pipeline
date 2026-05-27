# Async Order Pipeline

Pipeline **assíncrono** em Go para ingestão e processamento de webhooks de pedidos. Recebe eventos dos gateways **guren** e **lancelot**, decripta payloads AES-256-GCM, valida e deduplica via claim atômico, persiste em MySQL através de stored procedure transacional, e faz fanout para 4 canais de marketing (SMS, Email, Call Center, WhatsApp) via RabbitMQ. A API retorna rápido, persistência e distribuição rodam em background, desacopladas por filas e workers.

## Stack

- Go
- MySQL 8.0
- RabbitMQ 3 (management)
- Docker Compose

## Como subir

```bash
docker compose up -d --build
```

Tudo configurado no `docker-compose.yml`, sem necessidade de setup adicional. O `docker-compose.yml` sobe:

- **mysql** — porta `3306`, usuário/senha `mysql/mysql`, database `db`. Migrations em `sql/` rodam automaticamente.
- **rabbitmq** — AMQP em `5672`, UI de management em `15672` (usuário/senha `rabbit/rabbit`).
- **app** — servidor HTTP em `8080`, com 2 workers internos (lead consumer e SMS distributor).

O app só inicia após MySQL e RabbitMQ estarem saudáveis.

## Endpoints

| Método | Path | Descrição |
|---|---|---|
| `POST` | `/webhooks/guren` | Recebe webhook do gateway guren (payload criptografado AES-256-GCM se header `X-Guren-Encrypted: true`) |
| `POST` | `/webhooks/lancelot` | Recebe webhook do gateway lancelot (payload em JSON aberto) |
| `GET` | `/health` | Healthcheck |

## Configuração

Todas as variáveis estão definidas no `docker-compose.yml` para facilitar setup:

| Variável | Valor / Descrição |
|---|---|
| `PORT` | `8080` |
| `DSN` | Connection string MySQL |
| `RABBIT_URL` | URL AMQP do RabbitMQ |
| `GUREN_KEY` | Chave AES-256 em hex (64 chars) para decrypt do guren |
| `WEBHOOK_SITE_URL` | URL do webhook.site para entrega simulada de SMS (vazio = skip) |

## Estrutura

```
cmd/
  api/                  entry point HTTP (main, handlers, rotas)
  scripts/
    test-webhooks/      script que testa, envia os payloads e cruza com os dados do banco
    generate/           gerador do dataset de teste (encripta os guren com GCM)
data/
  webhook_payloads.json dataset de teste (20 payloads)
internal/
  config/               leitura de env vars
  crypto/               AES-256-GCM decrypt
  domain/               structs (WebhookPayload, LeadMessage, DistMessage, DLQMessage)
  queue/                cliente RabbitMQ
  repository/           acesso MySQL
  service/              validação e normalização
  worker/               lead consumer e SMS distributor
sql/                    migrations (tabelas, índices, stored procedure)
audit/                  queries de auditoria (audit_queries.sql)
```

## Rodar o teste end-to-end

```bash
go run ./cmd/scripts/test-webhooks
```

Envia os 20 payloads de `data/webhook_payloads.json` para a API e cruza as respostas HTTP com o estado canônico do banco, mostrando a distribuição final em 5 categorias.

## Verificar manualmente no banco

```bash
docker exec -it mysql mysql -u mysql -pmysql db
```

```sql
SELECT COUNT(*) FROM raw_payloads;
SELECT COUNT(*) FROM lead_events WHERE event = 'order.approved';
SELECT origin, COUNT(*) FROM lead_dead_letter GROUP BY origin;
SELECT channel, status, COUNT(*) FROM distribution_status GROUP BY channel, status;
```

RabbitMQ UI: <http://localhost:15672> (rabbit/rabbit).
