# AGENTS.md — ofertas-backoffice

Backoffice local em Vite + React + TypeScript para administrar entidades do domínio de ofertas.

## Escopo

- SPA com identificação do **Operador** (nome + senha; cookie httpOnly). Catálogo em `/api/...` (Vite/nginx proxy para `ofertas-api`). Canal WhatsApp em `/gateway/...` (proxy para `gateway-whatsapp`) — estado/Pareamento, Conversas, Mensagens e envio de texto. Glossário: [`CONTEXT.md`](./CONTEXT.md).
- UI administrativa funcional: tabelas, filtros, formulários, confirmações de exclusão e mensagens claras de erro da API. Superfície **Canal**: QR quando pendente, JID quando há Pareamento, Desparear com confirmação. Conversas são rastro só-leitura com compositor nas allowlisted. Operações: Testar Mercado faz scan de encarte ou scraping de site (sem Extrator).
- Use termos do glossário do scraper para o catálogo; para o canal, use [`../gateway-whatsapp/CONTEXT.md`](../gateway-whatsapp/CONTEXT.md): Contato, Conversa, Mensagem, Mídia, Allowlist, Canal, Pareamento, Desparear; **Operador** e **Sair** neste `CONTEXT.md`.

## Comandos

```bash
npm install
npm run dev
npm run build
npm test
```

No Compose: `docker compose up --build` (ADR 0008) publica o SPA em `http://localhost:5173` (nginx faz proxy de `/api` e `/gateway`). Host `npm run dev` usa o proxy do Vite. Não bakeie `GATEWAY_TOKEN` no SPA (ADR 0018).

GCP (Cloud Run, projeto `orquestrador-ofertas`, `southamerica-east1`): https://ofertas-backoffice-ot4vhmelqq-rj.a.run.app — [console](https://console.cloud.google.com/run/detail/southamerica-east1/ofertas-backoffice/metrics?project=orquestrador-ofertas).

## Regras locais

- O Operador identifica-se em `/entrar`. Não há CRUD de Operador na UI (insert/delete no Postgres). `GATEWAY_TOKEN` não entra no bundle.
- Mantenha o cliente de API tolerante a pequenas variações de payload, mas preserve os endpoints esperados em `/api`.
- Não comite `.env` nem valores secretos; versionar apenas `.env.example`.
- Prefira componentes simples e reutilizáveis antes de adicionar bibliotecas de UI.
- Siga TDD do workspace em [`../../AGENTS.md`](../../AGENTS.md): um comportamento observável — teste que falha → código que passa → refatorar o teste → refatorar o código.
- Vite 8 / Vitest 4 exigem o binding nativo do Rolldown. Neste Node, o npm pode pular o opcional; `package.json` pina `@rolldown/binding-linux-x64-gnu`. Noutro OS: `npm i -D @rolldown/binding-<plataforma>`.

## Exibição na UI

Listas e detalhes mostram **relacionamentos e valores legíveis**. Os payloads da API continuam CRUD fino (IDs, ISO, JSON); a formatação vive no backoffice (`src/display.ts`, `FormattedValue`, `useCatalogLookups`).

| Tipo | Como declarar | Exibição |
|------|----------------|----------|
| FK | `format: "ref"` + `ref` na coluna; `kind: "ref"` + `ref` no campo/filtro | Nome (Produto, Marca, Mercado), `id` (Fonte), `filename (dia)` (Documento); hover mostra o UUID; clique abre o detalhe |
| Data | `format: "date"` / `kind: "date"` | `dd/MM/yyyy` em `America/Sao_Paulo` |
| Instante | `format: "datetime"` / `kind: "datetime"` | `dd/MM/yyyy HH:mm` |
| Moeda | `format: "currency"` | `R$ 19,90` |
| Quantidades | `format: "quantidades"` | `500 / 1.000` (sem colchetes) |

Regras:

- Não altere o contrato da API para embeder nomes. Carregue catálogos via `useCatalogLookups()` (queries paralelas em `/mercados`, `/produtos`, `/marcas`, `/fontes`, `/documentos`).
- Declare `format` / `ref` em `resources.ts`. Páginas custom (`OperationsPage`, `UsoExtratorPages`, `FalhasExtracaoPage`) reutilizam `FormattedValue` e `RefSelect`.
- Formulários e filtros escolhem a entidade relacionada (select por rótulo) e enviam o ID. JSON/números crus ficam só nos campos que a API espera assim (`quantidades`, `promocao`).
- Se o registro da FK não existir mais, mostre `{id} (não encontrado)` sem link.
