# Context Map

## Contexts

- [Catálogo de ofertas](./apps/ofertas-scraper/CONTEXT.md) — Oferta, Produto, Mercado, Fonte, Documento, Coleta, Extrator, Indicação Promocional (encarte, ofertas-scraper-v2, API e backoffice)
- [Canal WhatsApp](./apps/gateway-whatsapp/CONTEXT.md) — Contato, Conversa, Mensagem, Mídia, Canal
- [Agente de ofertas](./apps/agente-ofertas/CONTEXT.md) — Lista, Item, Resposta; papéis Agente da Lista, Agente de Filtragem, Agente de Resposta (assistente de catálogo é superfície deste contexto, não um quarto)

Cadeia Lista → Coleta → Resposta: [`docs/arquitetura-agentes-lista-coleta.md`](./docs/arquitetura-agentes-lista-coleta.md). Testar Mercado (scan encarte / scraping site, sem Extrator): ADR [0015](./docs/adr/0015-testar-mercado-encarte-ou-site.md).

## Relationships

- **Catálogo ↛ Canal**: o gateway não lê nem escreve Oferta; estado de envio não entra nas tabelas do catálogo (workspace ADR 0006 / 0007)
- **Canal → backoffice**: o SPA lê Conversa/Mensagem (e envia texto na allowlist) via HTTP do `gateway-whatsapp` (ADR 0009), não via `ofertas-api`
- **Canal → Agente da Lista**: Mensagem viva de texto na Allowlist vira Lista; o gateway não chama Filtragem, Resposta nem o scraper
- **Agente da Lista → Coleta**: um Item, uma busca; Itens da mesma Lista em paralelo no `ofertas-scraper-v2`; a Coleta persiste e devolve os mesmos dados
- **Agente da Lista → Agente de Filtragem**: só com a flag de múltiplos tipos
- **Agente de Resposta → Canal**: a Resposta sai pelo Canal (`POST /envios`). Ack não dispara nessas Mensagens quando a cadeia está ligada
- **Encarte e Coleta**: dois caminhos de escrita no mesmo catálogo; Fonte/Documento não servem ao site; Coleta não substitui o Extrator. `ofertas-web-scraper` sai deste recorte
- **Assistente de catálogo**: entra no Agente da Lista sem WhatsApp — não é contexto à parte
