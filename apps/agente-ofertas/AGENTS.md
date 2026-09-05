# AGENTS.md — agente-ofertas

Agente de ofertas: interpreta uma **Lista** (Itens) contra o catálogo e envia a **Resposta** pelo Canal. Glossário: [`CONTEXT.md`](./CONTEXT.md). Workspace: [`../../AGENTS.md`](../../AGENTS.md). ADRs: [`../../docs/adr/0010-agente-ofertas-um-app.md`](../../docs/adr/0010-agente-ofertas-um-app.md), [`docs/adr/0001-notifica-http-casamento-por-substring.md`](./docs/adr/0001-notifica-http-casamento-por-substring.md), [`docs/adr/0002-resposta-so-mais-barato.md`](./docs/adr/0002-resposta-so-mais-barato.md).

Module path: `github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas`

## What this system does

1. Receives a Lista (`POST /listas` from the gateway, Bearer `GATEWAY_TOKEN`)
2. Parses Itens, matches Produto (substring on `nomeNorm`) and optional Marca
3. Reads Ofertas vigentes today (`America/Sao_Paulo`) via `modules/ofertas-store`
4. Sends the Resposta through gateway `POST /envios`
5. Exposes the same interpretation on `POST /interpretar` and `POST /mcp` (`interpretar_lista`) — catalog assistant, not WhatsApp

Does **not** pair WhatsApp, persist Mensagem, or run the Extrator.

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
