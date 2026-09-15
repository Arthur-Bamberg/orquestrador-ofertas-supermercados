# AGENTS.md — agente-ofertas

Dois papéis num processo (Agente da Lista, Agente de Resposta). Glossário: [`CONTEXT.md`](./CONTEXT.md). Cadeia: [`../../docs/arquitetura-agentes-lista-coleta.md`](../../docs/arquitetura-agentes-lista-coleta.md). Workspace: [`../../AGENTS.md`](../../AGENTS.md). ADRs: [`../../docs/adr/0010-agente-ofertas-um-app.md`](../../docs/adr/0010-agente-ofertas-um-app.md), [`../../docs/adr/0016-agente-lista-coleta-encarte-resposta.md`](../../docs/adr/0016-agente-lista-coleta-encarte-resposta.md), [`docs/adr/0001-notifica-http-casamento-por-substring.md`](./docs/adr/0001-notifica-http-casamento-por-substring.md), [`docs/adr/0002-resposta-so-mais-barato.md`](./docs/adr/0002-resposta-so-mais-barato.md), [`docs/adr/0003-termo-antes-da-coleta.md`](./docs/adr/0003-termo-antes-da-coleta.md), [`docs/adr/0004-resposta-um-produto-menos-extras.md`](./docs/adr/0004-resposta-um-produto-menos-extras.md), [`docs/adr/0005-intencao-antes-da-lista.md`](./docs/adr/0005-intencao-antes-da-lista.md).

Module path: `github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas`

## What this system does

1. **Agente da Lista** receives inbound text (`POST /listas` from the gateway, Bearer `GATEWAY_TOKEN`), classifies **Intenção** (Lista, Consulta, Recusa). Recusa/Consulta historically silent on `…@g.us`; the Canal recorte is direta. Consulta in a 1:1 is grounded on the Pague Menos Mercado description. Recusa in a 1:1 is a fixed recorte line. Lista: splits Itens, derives a Termo per Item (interpretation, closed-class fallback), runs Coleta per non-empty Termo in parallel via `ofertas-scraper-v2`, and reads vigente encarte Ofertas (Documento)
2. Waits for every Item, then **Agente de Resposta** drops related products, keeps the Produto with fewest extra name tokens, cheapest Oferta (price ties listed), and sends through gateway `POST /envios`
3. Catalog assistant (`POST /interpretar`, `POST /mcp`) enters at Agente da Lista — same Intenção and chain, no WhatsApp (Recusa still returns the recorte line). Both require Bearer `GATEWAY_TOKEN`.

Does **not** persist Mensagem or run the Extrator. Does **not** HTTP the supermarket itself (Coleta is `ofertas-scraper-v2`). WhatsApp transport is Cloud API on the gateway (ADR 0020).

## Stack

| Concern | Choice |
|---------|--------|
| Language | Go (workspace module) |
| Catalog | `modules/ofertas-store` (read) |
| Coleta | HTTP `COLETA_URL` → `ofertas-scraper-v2` `POST /coletas` |
| Termo / Intenção | Gemini (`GEMINI_API_KEY`); sem chave, o texto segue como Lista e o Termo cai no invólucro |
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
