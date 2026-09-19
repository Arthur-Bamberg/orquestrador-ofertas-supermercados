# Context Map

Nome comercial atual do produto: **Pague Menos Mercado**. Não é um Mercado do catálogo nem um Canal.

## Contexts

- [Catálogo de ofertas](./apps/ofertas-scraper/CONTEXT.md) — Oferta, Produto, Mercado, Fonte, Documento, Coleta, Extrator, Indicação Promocional (encarte, ofertas-scraper-v2, API e backoffice)
- [Canal WhatsApp](./apps/gateway-whatsapp/CONTEXT.md) — Contato (Boas-vindas, Pedido de Aceite, Aceite, Termos de Uso), Conversa, Mensagem, Mídia, Canal, Pareamento, Desparear
- [Agente de ofertas](./apps/agente-ofertas/CONTEXT.md) — Lista, Item, Termo, Resposta, Intenção (Consulta, Recusa); papéis Agente da Lista e Agente de Resposta (assistente de catálogo é superfície deste contexto, não um terceiro)
- [Backoffice](./apps/ofertas-backoffice/CONTEXT.md) — Operador (quem se identifica na SPA para administrar catálogo e Canal)

Cadeia Lista → Coleta → Resposta: [`docs/arquitetura-agentes-lista-coleta.md`](./docs/arquitetura-agentes-lista-coleta.md). Testar Mercado (scan encarte / scraping site, sem Extrator): ADR [0015](./docs/adr/0015-testar-mercado-encarte-ou-site.md). Cadeia do Agente: ADR [0016](./docs/adr/0016-agente-lista-coleta-encarte-resposta.md). Pareamento no backoffice: ADR [0017](./docs/adr/0017-backoffice-pareamento-via-gateway.md). Operador no backoffice: ADR [0018](./docs/adr/0018-operador-backoffice.md). Aceite no Contato no lugar da Allowlist como portão do produto: ADR [0019](./docs/adr/0019-aceite-no-contato-em-vez-de-allowlist.md). Canal always-on numa VM: ADR [0020](./docs/adr/0020-canal-always-on-na-vm.md).

## Relationships

- **Catálogo ↛ Canal**: o gateway não lê nem escreve Oferta; estado de envio não entra nas tabelas do catálogo (workspace ADR 0006 / 0007)
- **Operador → Catálogo e Canal**: só o Operador identificado administra no backoffice (catálogo via `ofertas-api`, rastro e Canal via `gateway-whatsapp`). Não é Contato. Não há cadastro na UI. Permanece identificado até Sair ou a linha ser apagada.
- **Máquina ↛ Operador**: chamadas entre processos identificam-se como máquina, não com nome e senha. Anônimo não chama HTTP de administração nem as portas entre apps.
- **Canal → backoffice**: o SPA lê Conversa/Mensagem (e envia texto na Allowlist), o estado do Canal, o QR quando pendente e o JID do Pareamento; Desparear parte da superfície Canal. Tudo via `gateway-whatsapp` (ADR 0009 / 0017), não via `ofertas-api`
- **Canal → Agente da Lista**: Mensagem viva de texto cujo Contato remetente tem Aceite entra no Agente; só é Lista se a Intenção for Lista. Sem Aceite o Canal envia Boas-vindas ou Pedido de Aceite e não chama o Agente. Consulta e Recusa em grupo não geram Mensagem de saída. O gateway não chama Resposta nem o scraper (ADR 0019)
- **Agente da Lista → Coleta**: um Termo por Item (vazio não busca); Itens da mesma Lista em paralelo no `ofertas-scraper-v2`; a Coleta persiste e devolve os mesmos dados; reuso no dia se já houver Coleta `concluido`/`parcial`
- **Agente da Lista → encarte**: lê Ofertas vigentes ligadas a Documento cujo Produto/Marca casa com o Termo; não lê Oferta que só existe por Coleta de outro dia
- **Agente da Lista → Agente de Resposta**: só depois de todas as Coletas e leituras de encarte da Lista; o organizador não busca. Interpreta o Item contra as Ofertas reunidas (tamanho, sabor, Marca, embalagem ditos restringem). No mesmo Produto e Mercado, Coleta do dia prevalece; encarte preenche o resto. Relacionados saem; Marca no Termo restringe; um Produto por Item (menos extras); empate de preço lista
- **Agente → Canal**: Resposta, e na direta Consulta e Recusa, saem por `POST /envios`. Em grupo só a Lista gera Mensagem de saída. Ack não dispara nessas Mensagens quando a cadeia está ligada
- **Encarte e Coleta**: dois caminhos de escrita no mesmo catálogo; Fonte/Documento não servem ao site; Coleta não substitui o Extrator. `ofertas-web-scraper` sai deste recorte
- **Assistente de catálogo**: entra no Agente da Lista sem WhatsApp — não é contexto à parte
