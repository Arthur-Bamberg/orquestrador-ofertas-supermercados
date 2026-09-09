# Arquitetura: Lista → Coleta → Resposta

Como uma Mensagem no Canal vira busca nos sites e Resposta. Glossário: [Catálogo](../apps/ofertas-scraper/CONTEXT.md), [Agente](../apps/agente-ofertas/CONTEXT.md), [Canal](../apps/gateway-whatsapp/CONTEXT.md), [mapa](../CONTEXT-MAP.md).

Três papéis de Agente, **um** app (`agente-ofertas`). Cada papel tem o próprio contrato de interpretação (prompt + input). A Coleta de vitrine neste recorte é o app `ofertas-scraper-v2` (Fort; loja = default do site). `ofertas-web-scraper` sai. Shopfully não entra.

## Papéis

| Papel | Quem chama | Faz | Não faz |
|-------|------------|-----|---------|
| **Agente da Lista** | Canal (Mensagem viva na Allowlist) | Parte a Lista em Itens; dispara **em paralelo** uma Coleta por Item nos sites; junta catálogo + dados consultados; liga a flag de múltiplos tipos quando couber | Não filtra relacionados; não envia WhatsApp |
| **Agente de Filtragem** | Agente da Lista, **só** com a flag de múltiplos tipos | Descarta relacionados (molho quando pediu tomate); devolve os tipos vendáveis e onde está o mais barato, olhando **a base e o que foi consultado** | Não busca no site; não envia WhatsApp |
| **Agente de Resposta** | Agente da Lista (com ou sem Filtragem no meio) | Monta a Resposta (bloco por Item) e chama o Canal (`POST /envios`) | Não busca; não persiste Oferta |

Sem flag (um tipo só): Lista → Coletas → Resposta. Com flag: Lista → Coletas → Filtragem → Resposta.

## Sequência

```
Mensagem viva (Allowlist)
        │
        ▼
Agente da Lista          ← gateway-whatsapp (não chama Filtragem nem Resposta direto)
        │
        ├─ Item 1 ──► ofertas-scraper-v2 (Coleta Fort) ─┐
        ├─ Item 2 ──► ofertas-scraper-v2 (Coleta Fort) ─┼─ paralelo
        └─ Item N ──► ofertas-scraper-v2 (Coleta Fort) ─┘
                            │
                            ▼
                   persiste todo cartão completo
                   como Oferta (Produto/Marca
                   match-or-create)
                            │
                            ▼
              dados consultados + catálogo (base)
                            │
              ┌─────────────┴──────────────┐
              │ flag múltiplos tipos?      │
              │ sim → Agente de Filtragem  │
              │ não → (pula)               │
              └─────────────┬──────────────┘
                            ▼
                   Agente de Resposta
                            │
                            ▼
              gateway-whatsapp POST /envios
                   (Resposta na Conversa)
```

O gateway **não** chama o scraper. O scraper **não** chama o Agente de Resposta. Quem orquestra é o Agente da Lista.

## Coleta (`ofertas-scraper-v2`)

- Termo da busca = texto do Item.
- Identidade: **termo + Mercado + dia** (`America/Sao_Paulo`). Concluído ou parcial no dia: não volta ao site, devolve o conjunto ao Agente da Lista (consultado = o que veio; Filtragem/Resposta não distinguem incompleto). `falhou` e `processando` órfão retentam (aí o conjunto é substituído). `processando` em voo: a segunda chamada espera o mesmo conjunto, não dispara outra busca. Caminho feliz: pesquisa até esgotar os cartões. Página seguinte falhou ou busca não terminou a tempo: persiste o que já paginou, marca parcial, devolve ao Agente da Lista (busca salva — não retenta no dia). Vitrine vazia esgotada: concluído sem Ofertas (salva). Queda sem Oferta: falhou.
- Persiste Oferta de **cada** cartão completo, inclusive relacionados. Produto é o tipo vendável (sem marca; tamanho na Oferta); Marca à parte quando o cartão trouxer.
- Devolve ao Agente da Lista os **mesmos dados** que gravou (não só um “foi”).
- Neste recorte: Mercado Fort (adapter no app; seed da Fonte `tipo=site` em `https://www.fortatacadista.com.br/` só tira o encarte do job). Loja = default do site. Outros sites entram como adapter no mesmo app, noutro recorte.

- Encarte (`ofertas-scraper`) continua outro caminho de escrita no mesmo catálogo.

## O que a Resposta mostra

Por Item:

- Relacionados ficam de fora (Filtragem quando há vários tipos; um tipo só não precisa dela).
- Vários tipos vendáveis: lista cada tipo com o **preço observado** no cartão — não preço unitário derivado, não um mínimo entre tipos.
- Mesmo tipo, várias Marcas no mesmo Mercado: lista cada Marca com o preço observado.
- **Onde está o mais barato**: cruza Ofertas já na base com o que a Coleta acabou de ver.

Zero tipos depois do filtro: diz que não achou.

## Apps

| App | Neste fluxo |
|-----|-------------|
| `gateway-whatsapp` | Entrega a Lista ao Agente da Lista; envia a Resposta (`POST /envios`). Não lê Oferta. |
| `agente-ofertas` | Os três papéis. Assistente de catálogo (MCP / `POST /interpretar`) entra no Agente da Lista sem Canal. |
| `ofertas-scraper-v2` | Coleta de vitrine (Fort neste recorte). |
| `ofertas-scraper` | Encarte (Fonte → Documento → Extrator). Fora desta cadeia. |
| `ofertas-web-scraper` | Removido; Carrefour/Asun/Rissul ficam sem preço de site até o recorte deles no v2. |

## Contratos entre papéis (mesmo processo)

1. **Canal → Agente da Lista:** texto da Mensagem (e JID da Conversa para a Resposta voltar).
2. **Agente da Lista → Coleta:** um termo (Item) por chamada; N chamadas em paralelo.
3. **Coleta → Agente da Lista:** cartões/Ofertas persistidos daquela busca.
4. **Agente da Lista → Agente de Filtragem** (se flag): Item + dados consultados + recorte da base. Flag = há múltiplos tipos vendáveis na base e/ou no consultado.
5. **Filtragem (ou Lista) → Agente de Resposta:** tipos que valem, preços observados, Mercado mais barato por tipo.
6. **Agente de Resposta → Canal:** texto da Resposta + JID.
