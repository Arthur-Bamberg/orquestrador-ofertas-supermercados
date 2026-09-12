# AGENTS.md — gateway-whatsapp

Gateway do canal WhatsApp. Glossário: [`CONTEXT.md`](./CONTEXT.md). Workspace: [`../../AGENTS.md`](../../AGENTS.md). ADRs: [`../../docs/adr/0007-gateway-whatsapp-whatsmeow.md`](../../docs/adr/0007-gateway-whatsapp-whatsmeow.md), [`../../docs/adr/0009-backoffice-le-canal-via-gateway.md`](../../docs/adr/0009-backoffice-le-canal-via-gateway.md), [`../../docs/adr/0017-backoffice-pareamento-via-gateway.md`](../../docs/adr/0017-backoffice-pareamento-via-gateway.md), [`docs/adr/0001-rastro-completo-whatsapp.md`](./docs/adr/0001-rastro-completo-whatsapp.md), [`docs/adr/0003-receber-concorrente-unique.md`](./docs/adr/0003-receber-concorrente-unique.md).

Module path: `github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp`

## What this system does

Long-running process that:

1. Pair via QR (whatsmeow) unless `WHATSAPP_STUB=1`
2. Accepts inbound text and media in 1:1, groups and Status (stories)
3. Persists Contato / Conversa / Mensagem in Postgres schema `whatsapp` (allowlist does **not** gate INSERT)
4. Stores media bytes under `MIDIA_ROOT` (best-effort; row is kept if download fails)
5. Sends a static ack only to allowlisted Conversas when `AGENTE_URL` is empty (no Oferta lookup). With `AGENTE_URL`, those live text Mensagens go to `agente-ofertas` instead of Ack
6. Exposes `GET /health`, `GET /ready`, `GET /conversas`, `GET /conversas/{id}/mensagens`, `GET /mensagens/{id}/midia` (rastro; no Bearer), `GET /canal` and `POST /canal/desparear` (Bearer `GATEWAY_TOKEN`; estado / QR / Desparear) and `POST /envios` (Bearer `GATEWAY_TOKEN`)

Does **not** import `modules/ofertas-store`. The Agente is `apps/agente-ofertas`.

## Stack

| Concern | Choice |
|---------|--------|
| Language | Go (workspace module) |
| WhatsApp | whatsmeow; session sqlite in `.data/` |
| Catalog | none |
| Channel DB | PostgreSQL schema `whatsapp` (same instance, own migrations) |
| HTTP | `:8090` default; `CORS_ORIGIN` for the backoffice |

## Layout

```
cmd/gateway-whatsapp/
internal/
  domain/
  application/
  httpapi/
  config/
  infra/pg  infra/midia  infra/canal  infra/whatsapp  infra/agente
```

`presentation` here is HTTP (`httpapi`) plus `cmd` wiring.

## Local

Env na **raiz do monorepo** (o mesmo `.env` do compose). O binário sobe diretórios até achar `.env`.

```bash
# na raiz
cp .env.example .env   # se ainda não existir; complete GATEWAY_TOKEN e WHATSAPP_ALLOWLIST
docker compose up --build
# QR e estado do Canal: tela Canal no backoffice (`GET /canal`). Logs: `docker compose logs -f gateway-whatsapp`
```

With `WHATSAPP_STUB=1`, no QR; `/ready` is up; `POST /envios` talks to the stub.

Live WhatsApp: `WHATSAPP_STUB=0`, dedicated number, scan the QR on **Canal** in the backoffice (or gateway logs). Opt-in tests only: `LIVE_WHATSAPP=1` (none in the default suite).

Allowlist: comma-separated E.164 and/or group JIDs (`120363…@g.us`). Matching strips device suffix and the Brazilian extra `9` after DDD. The list gates **ack and POST /envios**, not persistence or `GET /conversas`. Group messages are stored even if the group JID is not listed. The backoffice compositor only appears when the Conversa resumo has `permitido: true`.

History sync (`events.HistorySync`) is ingested into `whatsapp.mensagem` (`origem=historico`). Chunks already dispatched before this handler existed are gone unless you re-pair (new QR). Inspect:

```sql
SELECT c.jid, c.tipo, m.direcao, m.origem, m.corpo, m.criado_em
FROM whatsapp.mensagem m
JOIN whatsapp.conversa c ON c.id = m.conversa_id
ORDER BY m.criado_em DESC
LIMIT 50;
```

## Tests

```bash
go test ./...
```

Unit tests use an in-memory Canal and, for Postgres, embedded-postgres (`internal/infra/pg/pgtest`).
