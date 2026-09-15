# AGENTS.md — gateway-whatsapp

Gateway do canal WhatsApp. Glossário: [`CONTEXT.md`](./CONTEXT.md). Workspace: [`../../AGENTS.md`](../../AGENTS.md). ADRs: [`../../docs/adr/0007-gateway-whatsapp-whatsmeow.md`](../../docs/adr/0007-gateway-whatsapp-whatsmeow.md), [`../../docs/adr/0009-backoffice-le-canal-via-gateway.md`](../../docs/adr/0009-backoffice-le-canal-via-gateway.md), [`../../docs/adr/0017-backoffice-pareamento-via-gateway.md`](../../docs/adr/0017-backoffice-pareamento-via-gateway.md), [`../../docs/adr/0018-operador-backoffice.md`](../../docs/adr/0018-operador-backoffice.md), [`../../docs/adr/0019-aceite-no-contato-em-vez-de-allowlist.md`](../../docs/adr/0019-aceite-no-contato-em-vez-de-allowlist.md), [`../../docs/adr/0020-canal-always-on-na-vm.md`](../../docs/adr/0020-canal-always-on-na-vm.md), [`docs/adr/0001-rastro-completo-whatsapp.md`](./docs/adr/0001-rastro-completo-whatsapp.md), [`docs/adr/0003-receber-concorrente-unique.md`](./docs/adr/0003-receber-concorrente-unique.md).

Module path: `github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp`

## What this system does

Long-running process that:

1. Pair via QR (whatsmeow) unless `WHATSAPP_STUB=1`
2. Accepts inbound text and media in 1:1, groups and Status (stories)
3. Persists Contato / Conversa / Mensagem in Postgres schema `whatsapp` (allowlist does **not** gate INSERT)
4. Stores media bytes under `MIDIA_ROOT` (best-effort; row is kept if download fails)
5. First live text from a Contato without Boas-vindas gets the presentation + Termos de Uso (`1`/`2`). Later texts without Aceite get Pedido de Aceite. After Aceite: static ack when `AGENTE_URL` is empty; with `AGENTE_URL`, those Mensagens go to `agente-ofertas`. Allowlist does **not** gate this path. FromMe in a group skips the Aceite gate.
6. Exposes `GET /health`, `GET /ready` (open); `GET /conversas`, `GET /conversas/{id}/mensagens`, `GET /mensagens/{id}/midia`, `GET /canal`, `POST /canal/desparear` (Operador cookie); `POST /envios` (Operador cookie **or** Bearer `GATEWAY_TOKEN`)

Does **not** import `modules/ofertas-store`. The Agente is `apps/agente-ofertas`.

## Stack

| Concern | Choice |
|---------|--------|
| Language | Go (workspace module) |
| WhatsApp | whatsmeow; session sqlite in `.data/` |
| Catalog | none |
| Channel DB | PostgreSQL schema `whatsapp` (same instance, own migrations) |
| HTTP | `:8090` default; `CORS_ORIGIN` for the backoffice |
| Production | Always-on GCE VM `gateway-whatsapp` (e2-micro, `southamerica-east1-a`, http://34.39.249.109:8090). This process only (ADR 0020). Not Cloud Run. Startup: [`scripts/gce-startup.sh`](./scripts/gce-startup.sh). |

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

Allowlist: comma-separated E.164 and/or group JIDs (`120363…@g.us`). Matching strips device suffix and the Brazilian extra `9` after DDD. The list gates **only the Operador compositor** (`permitido` + `POST /envios` with Operador cookie). Bearer `GATEWAY_TOKEN` (Agente) and automatic Canal replies (Boas-vindas, Pedido de Aceite, Ack) ignore it. Persistence and `GET /conversas` were already ungated.

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
