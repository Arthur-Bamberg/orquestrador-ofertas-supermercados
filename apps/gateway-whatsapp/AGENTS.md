# AGENTS.md — gateway-whatsapp

Gateway do canal WhatsApp. Glossário: [`CONTEXT.md`](./CONTEXT.md). Workspace: [`../../AGENTS.md`](../../AGENTS.md). ADR: [`../../docs/adr/0007-gateway-whatsapp-whatsmeow.md`](../../docs/adr/0007-gateway-whatsapp-whatsmeow.md).

Module path: `github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp`

## What this system does

Long-running process that:

1. Pair via QR (whatsmeow) unless `WHATSAPP_STUB=1`
2. Accepts inbound text and media in allowlisted 1:1 and group chats
3. Persists Contato / Conversa / Mensagem in Postgres schema `whatsapp`
4. Stores media bytes under `MIDIA_ROOT`
5. Sends a static ack (no Oferta lookup)
6. Exposes `GET /health`, `GET /ready`, `POST /envios` (Bearer `GATEWAY_TOKEN`)

Does **not** import `modules/ofertas-store`. The agent worker is a later app.

## Stack

| Concern | Choice |
|---------|--------|
| Language | Go (workspace module) |
| WhatsApp | whatsmeow; session sqlite in `.data/` |
| Catalog | none |
| Channel DB | PostgreSQL schema `whatsapp` (same instance, own migrations) |
| HTTP | `:8090` default |

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
docker compose up -d
go run ./apps/gateway-whatsapp/cmd/gateway-whatsapp
```

With `WHATSAPP_STUB=1`, no QR; `/ready` is up; `POST /envios` talks to the stub.

Live WhatsApp: `WHATSAPP_STUB=0`, dedicated number, scan the QR printed on stderr. Opt-in tests only: `LIVE_WHATSAPP=1` (none in the default suite).

Allowlist: comma-separated E.164 and/or group JIDs (`120363…@g.us`). Group messages are dropped unless the **group** JID is listed.

## Tests

```bash
go test ./...
```

Unit tests use an in-memory Canal and, for Postgres, embedded-postgres (`internal/infra/pg/pgtest`).
