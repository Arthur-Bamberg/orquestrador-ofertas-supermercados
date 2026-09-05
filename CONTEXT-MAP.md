# Context Map

## Contexts

- [Catálogo de ofertas](./apps/ofertas-scraper/CONTEXT.md) — Oferta, Produto, Mercado, Fonte, Documento, Extrator (scraper, API e backoffice)
- [Canal WhatsApp](./apps/gateway-whatsapp/CONTEXT.md) — Contato, Conversa, Mensagem, Mídia, Canal
- [Agente de ofertas](./apps/agente-ofertas/CONTEXT.md) — Lista, Item, Resposta (assistente de catálogo é superfície deste contexto, não um quarto)

## Relationships

- **Catálogo ↛ Canal**: o gateway não lê nem escreve Oferta; estado de envio não entra nas tabelas do catálogo (workspace ADR 0006 / 0007)
- **Canal → backoffice**: o SPA lê Conversa/Mensagem (e envia texto na allowlist) via HTTP do `gateway-whatsapp` (ADR 0009), não via `ofertas-api`
- **Canal → Agente**: Mensagem viva de texto na Allowlist vira Lista; a Resposta sai pelo Canal (`POST /envios`). Ack não dispara nessas Mensagens quando o Agente está ligado.
- **Catálogo → Agente**: o Agente lê Oferta (e o catálogo); não escreve o catálogo nem chama o Extrator.
- **Assistente de catálogo**: mesma pergunta ao Agente, sem WhatsApp — não é contexto à parte.
