# AGENTS.md — agente-ofertas

Dois papéis num processo (Agente da Lista, Agente de Resposta). Glossário: [`CONTEXT.md`](./CONTEXT.md). Cadeia: [`../../docs/arquitetura-agentes-lista-coleta.md`](../../docs/arquitetura-agentes-lista-coleta.md). Workspace: [`../../AGENTS.md`](../../AGENTS.md). ADRs: [`../../docs/adr/0010-agente-ofertas-um-app.md`](../../docs/adr/0010-agente-ofertas-um-app.md), [`../../docs/adr/0016-agente-lista-coleta-encarte-resposta.md`](../../docs/adr/0016-agente-lista-coleta-encarte-resposta.md), [`docs/adr/0001-notifica-http-casamento-por-substring.md`](./docs/adr/0001-notifica-http-casamento-por-substring.md), [`docs/adr/0002-resposta-so-mais-barato.md`](./docs/adr/0002-resposta-so-mais-barato.md).

Module path: `github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas`

## What this system does

1. **Agente da Lista** receives a Lista (`POST /listas` from the gateway, Bearer `GATEWAY_TOKEN`), splits Itens, runs Coleta per Item in parallel via `ofertas-scraper-v2`, and reads vigente encarte Ofertas (Documento)
2. Waits for every Item, then **Agente de Resposta** drops related products, keeps vendable types, cheapest Oferta per type, and sends through gateway `POST /envios`
3. Catalog assistant (`POST /interpretar`, `POST /mcp`) enters at Agente da Lista — same chain, no WhatsApp

Does **not** pair WhatsApp, persist Mensagem, or run the Extrator. Does **not** HTTP the supermarket itself (Coleta is `ofertas-scraper-v2`).

## Stack

| Concern | Choice |
|---------|--------|
| Language | Go (workspace module) |
| Catalog | `modules/ofertas-store` (read) |
| Coleta | HTTP `COLETA_URL` → `ofertas-scraper-v2` `POST /coletas` |
| Canal | HTTP to `gateway-whatsapp` |
| HTTP | `:8091` default |

## Local

Env na **raiz** (Compose). `AGENTE_URL` no gateway aponta a este processo. `COLETA_URL` aponta ao scraper-v2.

```bash
docker compose up --build
# health: curl http://localhost:8091/health
```

## Tests

```bash
go test ./...
```

Unit tests use an in-memory catalog and a stub Coleta. No live WhatsApp, no live supermarket HTTP.
