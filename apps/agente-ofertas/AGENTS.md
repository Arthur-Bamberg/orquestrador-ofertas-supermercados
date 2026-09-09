# AGENTS.md — agente-ofertas

Três papéis num processo (Agente da Lista, Agente de Filtragem, Agente de Resposta). Glossário: [`CONTEXT.md`](./CONTEXT.md). Cadeia: [`../../docs/arquitetura-agentes-lista-coleta.md`](../../docs/arquitetura-agentes-lista-coleta.md). Workspace: [`../../AGENTS.md`](../../AGENTS.md). ADRs: [`../../docs/adr/0010-agente-ofertas-um-app.md`](../../docs/adr/0010-agente-ofertas-um-app.md), [`docs/adr/0001-notifica-http-casamento-por-substring.md`](./docs/adr/0001-notifica-http-casamento-por-substring.md), [`docs/adr/0002-resposta-so-mais-barato.md`](./docs/adr/0002-resposta-so-mais-barato.md).

Module path: `github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas`

## What this system does

1. **Agente da Lista** receives a Lista (`POST /listas` from the gateway, Bearer `GATEWAY_TOKEN`), splits Itens, runs Coleta per Item in parallel via `ofertas-scraper-v2`
2. Sets the multiple-types flag from catalog + consulted data; if set, **Agente de Filtragem** drops related products and names types + cheapest
3. **Agente de Resposta** sends the Resposta through gateway `POST /envios`
4. Catalog assistant (`POST /interpretar`, `POST /mcp`) enters at Agente da Lista — same chain, no WhatsApp

Does **not** pair WhatsApp, persist Mensagem, or run the Extrator. Does **not** HTTP the supermarket itself (Coleta is `ofertas-scraper-v2`).

## Stack

| Concern | Choice |
|---------|--------|
| Language | Go (workspace module) |
| Catalog | `modules/ofertas-store` (read) |
| Canal | HTTP to `gateway-whatsapp` |
| HTTP | `:8091` default |

## Local

Env na **raiz** (Compose). `AGENTE_URL` no gateway aponta a este processo.

```bash
docker compose up --build
# health: curl http://localhost:8091/health
```

## Tests

```bash
go test ./...
```

Unit tests use an in-memory catalog. No live WhatsApp, no live Extrator.
