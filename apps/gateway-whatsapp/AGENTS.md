# AGENTS.md — gateway-whatsapp

Gateway do canal WhatsApp. Glossário: [`CONTEXT.md`](./CONTEXT.md). Workspace: [`../../AGENTS.md`](../../AGENTS.md). ADRs: [`../../docs/adr/0007-gateway-whatsapp-whatsmeow.md`](../../docs/adr/0007-gateway-whatsapp-whatsmeow.md), [`../../docs/adr/0009-backoffice-le-canal-via-gateway.md`](../../docs/adr/0009-backoffice-le-canal-via-gateway.md), [`docs/adr/0001-rastro-completo-whatsapp.md`](./docs/adr/0001-rastro-completo-whatsapp.md), [`docs/adr/0002-resposta-automatica-fora-da-allowlist.md`](./docs/adr/0002-resposta-automatica-fora-da-allowlist.md).

Module path: `github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp`

## What this system does

Long-running process that:

1. Pair via QR (whatsmeow) unless `WHATSAPP_STUB=1`
2. Accepts inbound text and media in 1:1, groups and Status (stories)
3. Persists Contato / Conversa / Mensagem in Postgres schema `whatsapp` (allowlist does **not** gate INSERT)
4. Stores media bytes under `MIDIA_ROOT` (best-effort; row is kept if download fails)
5. Sends a static ack only to allowlisted Conversas (no Oferta lookup); **Resposta automática** (`WHATSAPP_AUTO_NOME` / `WHATSAPP_AUTO_TEXTO`) may send outside the allowlist and replaces the ack when both match
6. Exposes `GET /health`, `GET /ready`, `GET /conversas`, `GET /conversas/{id}/mensagens`, `GET /mensagens/{id}/midia` (rastro; no Bearer) and `POST /envios` (Bearer `GATEWAY_TOKEN`)

Does **not** import `modules/ofertas-store`. The agent worker is a later app.

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
  infra/pg  infra/midia  infra/canal  infra/whatsapp
```

`presentation` here is HTTP (`httpapi`) plus `cmd` wiring.

## Local

Env na **raiz do monorepo** (o mesmo `.env` do compose). O binário sobe diretórios até achar `.env`.

```bash
# na raiz
cp .env.example .env   # se ainda não existir; complete GATEWAY_TOKEN e WHATSAPP_ALLOWLIST
docker compose up --build
# QR (WHATSAPP_STUB=0): docker compose logs -f gateway-whatsapp
```

With `WHATSAPP_STUB=1`, no QR; `/ready` is up; `POST /envios` talks to the stub.

Live WhatsApp: `WHATSAPP_STUB=0`, dedicated number, scan the QR in the gateway logs (first pairing: `docker compose up` in the foreground so the QR renders). Opt-in tests only: `LIVE_WHATSAPP=1` (none in the default suite).

Allowlist: comma-separated E.164 and/or group JIDs (`120363…@g.us`). Matching strips device suffix and the Brazilian extra `9` after DDD. The list gates **ack and POST /envios**, not persistence or `GET /conversas`. **Resposta automática** (substring no nome da Conversa) envia mesmo fora da lista; ver ADR 0002. Group messages are stored even if the group JID is not listed. The backoffice compositor only appears when the Conversa resumo has `permitido: true`.

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
