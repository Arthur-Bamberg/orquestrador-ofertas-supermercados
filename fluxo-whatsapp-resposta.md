# Fluxo: WhatsApp → Resposta

Como uma Mensagem viva no Canal vira texto de volta no WhatsApp. Glossário: [Catálogo](./apps/ofertas-scraper/CONTEXT.md), [Canal](./apps/gateway-whatsapp/CONTEXT.md), [Agente](./apps/agente-ofertas/CONTEXT.md), [mapa](./CONTEXT-MAP.md). Contratos entre papéis: [`docs/arquitetura-agentes-lista-coleta.md`](./docs/arquitetura-agentes-lista-coleta.md).

O gateway **não** lê Oferta. O scraper-v2 **não** manda WhatsApp. Quem orquestra é `agente-ofertas` (dois papéis, um processo).

## Onde cada peça roda

| App | Papel neste fluxo | Onde |
|-----|-------------------|------|
| `gateway-whatsapp` | Recebe e envia WhatsApp; persiste rastro; porta Aceite | VM GCE always-on (`:8090`) |
| `agente-ofertas` | Classifica Intenção; dispara Coleta; lê encarte; monta Resposta | Cloud Run (`:8091`) |
| `ofertas-scraper-v2` | Busca vitrine e grava Oferta | Cloud Run (`:8092`) |
| `ofertas-scraper` | Encarte (Fonte → Documento → Extrator). **Fora** desta cadeia: só alimenta o catálogo | Job diário |

Postgres compartilhado. Canal usa o schema `whatsapp`. Catálogo passa por `modules/ofertas-store`. Chamadas máquina: Bearer `GATEWAY_TOKEN`.

## Cadeia

```
WhatsApp texto vivo
  → gateway persiste (schema whatsapp)
  → Boas-vindas / Pedido de Aceite / Aceite (Contato)
  → POST /listas
      → Gemini: Intenção
      → se Lista: Itens → Termo
          → POST /coletas (N em paralelo, reuso do dia)
          → lê encarte vigente (Documento)
      → espera tudo
      → escolhe Produto / Oferta
  → POST /envios
  → WhatsApp
```

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

## 1. WhatsApp chega no Canal

whatsmeow na VM (Pareamento por QR na tela Canal do backoffice). Cada evento `Message` vira um `Receber` em goroutine:

1. Dedup por id do provedor.
2. Upsert de **Contato** (JID de telefone) e **Conversa** (direta, grupo `…@g.us` ou Status).
3. Insert da **Mensagem** (`origem=vivo`). Mídia vai para disco se o download der; o conteúdo **não** é interpretado.

Histórico, Status, reação, revogação, indecifrável, FromMe em 1:1 e mídia sem caption **persistem**, mas **não** entram no produto.

A Allowlist **não** porta isso. Só vale para o compositor do Operador no backoffice.

## 2. Portão: Boas-vindas e Aceite (ainda no gateway)

Depois de persistir, só texto vivo (e FromMe em grupo que não seja eco do próprio Canal) passa pelo portão do **Contato**:

| Condição | Ação |
|----------|------|
| Sem Boas-vindas | Envia apresentação + Termos de Uso + “1 ou 2”. Nessa Mensagem o `1` ainda não vale |
| Sem Aceite, corpo = `1` | Grava Aceite e confirma |
| Sem Aceite, qualquer outro corpo | Pedido de Aceite (não republica os Termos) |
| Com Aceite | Chama o Agente (ou Ack se `AGENTE_URL` estiver vazio) |

Sem Aceite o texto **não** é Lista, Consulta nem Recusa. O Agente não é chamado.

## 3. Canal → Agente da Lista

Com Aceite e `AGENTE_URL` ligado:

```
POST {AGENTE_URL}/listas
Authorization: Bearer {GATEWAY_TOKEN}

{"conversaJid":"<JID da Conversa>","corpo":"<texto>"}
```

Timeout 5 min. O Agente responde `202` quando aceita o trabalho.

Se `AGENTE_URL` estiver vazio, sai o Ack estático (`WHATSAPP_ACK_TEXTO`). Com Agente ligado, **não** há Ack de fallback se a cadeia falhar.

## 4. Intenção

O Agente classifica **uma** Intenção por Mensagem (não acumula na Conversa). Com `GEMINI_API_KEY`, Gemini decide; sem chave (ou se o classificador falhar), o texto inteiro é **Lista**.

| Intenção | Direta | Grupo |
|----------|--------|--------|
| **Consulta** (saudação / “o que é isso?”) | Texto grounded na descrição do Pague Menos Mercado | Silêncio |
| **Recusa** (sai do recorte, inclusive se também pedir compra) | Linha fixa de recorte | Silêncio |
| **Lista** (quer comprar ou comparar preço) | Cadeia abaixo | Mesma Resposta **no grupo** |

## 5. Lista → Itens → Termo

1. Parte o texto em **Itens** (vírgula, `;`, quebra de linha, ` e `).
2. Para cada Item, deriva um **Termo** (tipo vendável + cultivar/processo + Marca se a pessoa disse). Gemini primeiro; se falhar, tira invólucro (verbo de compra, cumprimento, número, medida, artigo).
3. Termo vazio: aquele Item **não** busca; a Resposta dele é “Não achei.”

## 6. Coleta em paralelo + leitura de encarte

Para cada Termo não vazio, em paralelo:

```
POST {COLETA_URL}/coletas
Authorization: Bearer {GATEWAY_TOKEN}

{"termo":"..."}
```

Sem `mercadoId` → todos os adapters de vitrine (Fort, Stok, Bourbon, Carrefour).

Identidade da Coleta: **termo + Mercado + dia** (`America/Sao_Paulo`).

- Já `concluido` / `parcial` no dia → não volta ao site; devolve o conjunto.
- `falhou` / `processando` órfão → retenta.
- `processando` em voo → a segunda chamada espera a mesma, não dispara outra.
- Cada cartão completo vira Oferta (Produto/Marca match-or-create). Relacionados também são gravados; quem descarta é a Resposta.

Em paralelo, o Agente da Lista lê Ofertas **vigentes** ligadas a **Documento** (encarte do job `ofertas-scraper`) cujo Produto/Marca casa com o Termo. Zero casamentos no encarte **não** cancelam a Coleta.

## 7. Agente de Resposta (só depois de tudo)

Por Item:

- Relacionados saem (molho quando pediu tomate).
- Marca no Termo restringe; sem Oferta dela → não achou.
- Um Produto: o de **menos tokens a mais** no nome. Empate de extras → menor preço efetivo; empate desse preço lista esses Produtos.
- Em cada Produto, a Oferta mais barata (Marca e tamanho na linha; empate de preço lista todas).
- No mesmo Produto+Mercado: **Coleta do dia** se trouxe esse Produto; encarte vigente cobre Mercado sem Coleta do dia ou Produto que a Coleta não trouxe. Oferta só de Coleta de outro dia **não** entra.

Texto: um bloco por Item (título = o que a pessoa escreveu), com Mercado — Marca — preço / tamanho.

## 8. De volta ao WhatsApp

```
POST {gateway}/envios
Authorization: Bearer {GATEWAY_TOKEN}

{"conversaJid":"...","corpo":"..."}
```

Sem Allowlist. O gateway persiste a Mensagem de saída (`pendente` → `enviado`) e manda via whatsmeow. Recibos depois viram entregue/lido.

Consulta e Recusa na **direta** também saem por `/envios`. Em **grupo**, só Lista gera saída.

## Fora do WhatsApp

`POST /interpretar` e `POST /mcp` entram no **mesmo** Agente da Lista e devolvem o texto no HTTP, sem `POST /envios`.
