# Context Map

## Contexts

- [Catálogo de ofertas](./apps/ofertas-scraper/CONTEXT.md) — Oferta, Produto, Mercado, Fonte, Documento, Extrator (scraper, API e backoffice)
- [Canal WhatsApp](./apps/gateway-whatsapp/CONTEXT.md) — Contato, Conversa, Mensagem, Mídia, Canal

## Relationships

- **Catálogo ↛ Canal**: o gateway não lê nem escreve Oferta; estado de envio não entra nas tabelas do catálogo (workspace ADR 0006 / 0007)
- **Canal → backoffice**: o SPA lê Conversa/Mensagem (e envia texto na allowlist) via HTTP do `gateway-whatsapp` (ADR 0009), não via `ofertas-api`
- **Canal → agente (planejado)**: Mensagem persistida e `POST /envios` são o contrato; `agente-ofertas-worker` ainda não existe
- **MCP (planejado)**: acesso ao catálogo, não ao WhatsApp
