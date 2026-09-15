# AGENTS.md — gateway-whatsapp

Gateway do canal WhatsApp. Glossário: [`CONTEXT.md`](./CONTEXT.md). Workspace: [`../../AGENTS.md`](../../AGENTS.md). ADRs: [`../../docs/adr/0020-gateway-whatsapp-cloud-api.md`](../../docs/adr/0020-gateway-whatsapp-cloud-api.md), [`../../docs/adr/0007-gateway-whatsapp-whatsmeow.md`](../../docs/adr/0007-gateway-whatsapp-whatsmeow.md), [`../../docs/adr/0009-backoffice-le-canal-via-gateway.md`](../../docs/adr/0009-backoffice-le-canal-via-gateway.md), [`../../docs/adr/0018-operador-backoffice.md`](../../docs/adr/0018-operador-backoffice.md), [`../../docs/adr/0019-aceite-no-contato-em-vez-de-allowlist.md`](../../docs/adr/0019-aceite-no-contato-em-vez-de-allowlist.md), [`docs/adr/0001-rastro-completo-whatsapp.md`](./docs/adr/0001-rastro-completo-whatsapp.md), [`docs/adr/0003-receber-concorrente-unique.md`](./docs/adr/0003-receber-concorrente-unique.md).

Module path: `github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp`

## What this system does

HTTP process that:

1. Accepts Meta Cloud API webhooks (`GET /webhook` verify, `POST /webhook` signed) unless `WHATSAPP_STUB=1` (stub still parses webhooks; send does not call Graph)
2. Maps inbound user messages in **direta** to `Receber()` — Postgres, Aceite, Agente. Group and Status are not ingested
3. Persists Contato / Conversa / Mensagem in Postgres schema `whatsapp` (allowlist does **not** gate INSERT)
4. Stores media bytes under `MIDIA_ROOT` (best-effort download from Graph; row is kept if download fails)
5. First live text from a Contato without Boas-vindas gets the presentation + Termos de Uso (`1`/`2`). Later texts without Aceite get Pedido de Aceite. After Aceite: static ack when `AGENTE_URL` is empty; with `AGENTE_URL`, those Mensagens go to `agente-ofertas`. Allowlist does **not** gate this path
6. Sends via Graph HTTP (`POST /{phone-number-id}/messages`). Free-form text/media only inside the 24h Janela; Template outside it
7. Exposes `GET /health`, `GET /ready` (open; ready = Canal **pronto**); `GET /webhook`, `POST /webhook` (open, Meta); `GET /conversas`, `GET /conversas/{id}/mensagens`, `GET /mensagens/{id}/midia`, `GET /canal` (Operador cookie); `POST /envios` (Operador cookie **or** Bearer `GATEWAY_TOKEN`)

Does **not** import `modules/ofertas-store`. The Agente is `apps/agente-ofertas`.

## Stack

| Concern | Choice |
|---------|--------|
| Language | Go (workspace module) |
| WhatsApp | Cloud API (Graph HTTP + webhook); stub with `WHATSAPP_STUB=1` |
| Catalog | none |
| Channel DB | PostgreSQL schema `whatsapp` (same instance, own migrations) |
| HTTP | `:8090` default; `CORS_ORIGIN` for the backoffice |
| Deploy | Cloud Run scale-to-zero (ADR 0020), same shape as `ofertas-api` |

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
cp .env.example .env   # GATEWAY_TOKEN, WHATSAPP_ALLOWLIST; Cloud API: token, phone-number-id, verify, app secret
docker compose up --build
# estado do Canal: tela Canal no backoffice (`GET /canal`). Logs: `docker compose logs -f gateway-whatsapp`
```

With `WHATSAPP_STUB=1`, Graph is not called; `/ready` is up; `POST /envios` talks to the stub. Webhook POST still feeds `Receber()`.

Live Cloud API: `WHATSAPP_STUB=0`, `WHATSAPP_TOKEN`, `WHATSAPP_PHONE_NUMBER_ID`, `WHATSAPP_VERIFY_TOKEN`, `WHATSAPP_APP_SECRET`. Point Meta’s webhook to `https://<gateway>/webhook`. Opt-in tests only: `LIVE_WHATSAPP=1` (none in the default suite).

Allowlist: comma-separated E.164. Matching strips the Brazilian extra `9` after DDD. The list gates **only the Operador compositor** (`permitido` + `POST /envios` with Operador cookie). Bearer `GATEWAY_TOKEN` (Agente) and automatic Canal replies (Boas-vindas, Pedido de Aceite, Ack) ignore it. Persistence and `GET /conversas` were already ungated.

Free-form `POST /envios` requires an open Janela (last live inbound in that Conversa within 24h). Outside it, send a Template (`template.nome`). Inspect:

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

Unit tests use an in-memory Canal and, for Postgres, embedded-postgres (`internal/infra/pg/pgtest`). Graph and webhook use `httptest` (no live Meta).
