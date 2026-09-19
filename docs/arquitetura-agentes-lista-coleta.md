# Arquitetura: Lista → Coleta → Resposta

Como uma Mensagem no Canal vira busca nos sites, leitura de encarte e Resposta. Glossário: [Catálogo](../apps/ofertas-scraper/CONTEXT.md), [Agente](../apps/agente-ofertas/CONTEXT.md), [Canal](../apps/gateway-whatsapp/CONTEXT.md), [mapa](../CONTEXT-MAP.md).

Dois papéis de Agente, **um** app (`agente-ofertas`). A Coleta de vitrine é o app `ofertas-scraper-v2`. `ofertas-web-scraper` sai. Shopfully não entra.

## Papéis

| Papel | Quem chama | Faz | Não faz |
|-------|------------|-----|---------|
| **Agente da Lista** | Canal (Mensagem viva com Aceite no Contato remetente) ou assistente | Classifica a Intenção; se Lista, parte em Itens; obtém o Termo de cada Item; dispara **em paralelo** uma Coleta por Termo não vazio; lê Ofertas vigentes de encarte (Documento); espera todas as Coletas e o encarte | Não envia WhatsApp; não escolhe Produto |
| **Agente de Resposta** | Agente da Lista, só depois de todas as consultas | Organiza a Resposta (interpreta o Item contra as Ofertas reunidas; relacionados fora; Marca no Termo restringe; mais barato entre as que atendem; empate de preço lista) e chama o Canal (`POST /envios`) | Não busca; não persiste Oferta; não deriva o Termo; não consulta o catálogo por conta própria |

Cadeia: Intenção; se Lista → Coletas + encarte → Resposta. Não há Agente de Filtragem.

## Sequência

```
Mensagem viva de texto
        │
        ├─ sem Boas-vindas → Boas-vindas (Canal; não chama Agente; não lê `1`)
        └─ com Boas-vindas
                ├─ sem Aceite, corpo ≠ `1` → Pedido de Aceite (Canal)
                ├─ sem Aceite, corpo `1` → Aceite + confirmação (Canal)
                └─ com Aceite
                        ▼
Agente da Lista          ← gateway-whatsapp (não chama Resposta nem o scraper)
        │
        ├─ Recusa  → direta: texto de Recusa; grupo: nada
        ├─ Consulta → direta: texto de Consulta; grupo: nada
        └─ Lista
            ├─ Item 1 → Termo ──► ofertas-scraper-v2 (Coleta) ─┐
            ├─ Item 2 → Termo ──► ofertas-scraper-v2 (Coleta) ─┼─ paralelo
            └─ Item N → Termo ──► ofertas-scraper-v2 (Coleta) ─┘
        │                    │
        │                    ▼
        │           persiste todo cartão completo
        │           como Oferta (Produto/Marca
        │           match-or-create)
        │
        ├─ encarte vigente (Documento) ─────────────┐
        │                                            │
        ▼                                            ▼
              espera todas as Coletas e o encarte
                            │
                            ▼
                   Agente de Resposta
                            │
                            ▼
              gateway-whatsapp POST /envios
                   (Resposta na Conversa)
```

O gateway **não** chama o scraper. O scraper **não** chama o Agente de Resposta. Quem orquestra é o Agente da Lista.

Zero ou vários casamentos no encarte **não** cancelam a Coleta.

## Coleta (`ofertas-scraper-v2`)

- Termo da busca = Termo derivado do Item (não o texto cru).
- Identidade: **termo + Mercado + dia** (`America/Sao_Paulo`). Concluído ou parcial no dia: não volta ao site, devolve o conjunto ao Agente da Lista. `falhou` e `processando` órfão retentam (aí o conjunto é substituído). `processando` em voo: a segunda chamada espera o mesmo conjunto, não dispara outra busca. Caminho feliz: pesquisa até esgotar os cartões. Página seguinte falhou ou busca não terminou a tempo: persiste o que já paginou, marca parcial, devolve ao Agente da Lista (busca salva — não retenta no dia). Vitrine vazia esgotada: concluído sem Ofertas (salva). Queda sem Oferta: falhou.
- Persiste Oferta de **cada** cartão completo, inclusive relacionados. Produto é o tipo vendável (sem marca; tamanho na Oferta); Marca à parte quando o cartão trouxer.
- Devolve ao Agente da Lista os **mesmos dados** que gravou (não só um “foi”).
- `POST /coletas` sem `mercadoId` busca em todos os adapters de vitrine.

- Encarte (`ofertas-scraper`) continua outro caminho de escrita no mesmo catálogo.

## O que a Resposta mostra

Por Item:

- Relacionados ficam de fora (molho quando pediu tomate). Quem decide é o Agente de Resposta ao interpretar o Item, não substring no catálogo.
- Tamanho, sabor, Marca e embalagem no texto do Item restringem; Oferta mais barata que falha isso sai.
- Marca no Termo restringe a essa Marca; sem Oferta dela, não achou.
- Um Produto por Item entre os que atendem: o de menos tokens a mais no nome em relação ao tipo do Termo. Empate de extras → menor preço efetivo; empate desse preço lista esses Produtos. Em cada um, a Oferta de menor preço efetivo (Marca e tamanho na linha; empate de preço lista todas). Não preço unitário derivado, não todas as redes.
- No mesmo Produto e Mercado: vale a Oferta da **Coleta do dia** se essa Coleta trouxe esse Produto; encarte vigente cobre Mercado sem Coleta do dia (falhou, sem adapter) ou Produto que a Coleta não trouxe. Oferta que só existe por Coleta de outro dia não entra.

Zero tipos depois disso: diz que não achou.

## Apps

| App | Neste fluxo |
|-----|-------------|
| `gateway-whatsapp` | Entrega o texto da Mensagem ao Agente da Lista; envia a Resposta (`POST /envios`). Não lê Oferta. |
| `agente-ofertas` | Os dois papéis. Assistente de catálogo (MCP / `POST /interpretar`) entra no Agente da Lista sem Canal — mesma cadeia. |
| `ofertas-scraper-v2` | Coleta de vitrine. |
| `ofertas-scraper` | Encarte (Fonte → Documento → Extrator). Fora desta cadeia. |
| `ofertas-web-scraper` | Removido. |

## Contratos entre papéis (mesmo processo)

1. **Canal → Agente da Lista:** só se o Contato remetente tem Aceite; texto da Mensagem (e JID da Conversa para a saída voltar). O Agente da Lista classifica a Intenção. Sem Aceite o Canal não chama o Agente.
2. **Agente da Lista → Coleta:** só se Intenção for Lista; um Termo por Item (vazio não chama); N chamadas em paralelo.
3. **Coleta → Agente da Lista:** cartões/Ofertas persistidos daquela busca.
4. **Agente da Lista → encarte:** Ofertas vigentes ligadas a Documento cujo Produto/Marca casa com o tipo do Termo (sem o tamanho).
5. **Agente da Lista → Agente de Resposta:** conjunto reunido (consultado + encarte) e o texto de cada Item, só depois de todas as Coletas e do encarte da Lista.
6. **Agente de Resposta → Canal:** texto da Resposta + JID. Consulta e Recusa na direta também saem por `POST /envios`; em grupo não.
