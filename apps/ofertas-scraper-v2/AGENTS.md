# AGENTS.md — ofertas-scraper-v2

Coleta de vitrine: termo do Item → busca no site → Oferta de cada cartão completo. Glossário: [`../ofertas-scraper/CONTEXT.md`](../ofertas-scraper/CONTEXT.md). Cadeia: [`../../docs/arquitetura-agentes-lista-coleta.md`](../../docs/arquitetura-agentes-lista-coleta.md). Workspace: [`../../AGENTS.md`](../../AGENTS.md). ADRs: [`../../docs/adr/0012-coleta-termo-mercado-dia.md`](../../docs/adr/0012-coleta-termo-mercado-dia.md), [`../../docs/adr/0013-ofertas-scraper-v2-vitrine.md`](../../docs/adr/0013-ofertas-scraper-v2-vitrine.md), [`../../docs/adr/0014-coleta-reusa-mesmo-dia.md`](../../docs/adr/0014-coleta-reusa-mesmo-dia.md), [`../../docs/adr/0015-testar-mercado-encarte-ou-site.md`](../../docs/adr/0015-testar-mercado-encarte-ou-site.md). Novo Mercado / adapter de vitrine: [`.cursor/skills/vitrine-coleta/SKILL.md`](../../.cursor/skills/vitrine-coleta/SKILL.md).

Module path: `github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2`

## What this system does

HTTP (and one-shot CLI). Unit of work is **one Item term** (`POST /coletas` or `coletar <termo>`):

1. Search Fort (loja Sense id `1585` = site default Balneário Camboriú `115`) with the term; Sense page size 12 + `X-Osuper-Search-Token` from the Fort home page
2. Persist a **Coleta** (termo + Mercado + day) and **every** complete card as Oferta (Produto/Marca match-or-create)
3. Return the same Ofertas to the caller (Agente da Lista)
4. Same-day Coleta: reuse `concluido`/`parcial` (return persisted Ofertas to Agente da Lista, no site hit); retry `falhou` and orphan `processando` (then replace the set). In-flight `processando`: wait and share the same set — do not start a second Fort search. Happy path: search until the vitrine has no more cards. Cut-off / search that does not finish in time: persist what was already paged, mark `parcial`, return that set (saved — no same-day retry). Unmapped sale unit → `unidade`.
5. `POST /coletas` body: `{ "termo", "mercadoId"? }` — omit `mercadoId` to search every vitrine adapter; set it to restrict to one Mercado.

Does **not** filter related products (that's Agente de Resposta). Does **not** run the Extrator. Shopfully is out of this app.

## Stack

| Concern | Choice |
|---------|--------|
| Language | Go |
| Persistence | `modules/ofertas-store` (shared Postgres) |
| HTTP | `:8092` default, Bearer `GATEWAY_TOKEN` |
| Schedule | Request-driven from Agente da Lista; CLI for seed/ad-hoc |

## Layout

```
cmd/ofertas-scraper-v2/
internal/
  domain/
  application/
  presentation/
  infra/vitrine/fort
seed/mercados.json
```

## Local

```bash
# from this directory, with root .env DATABASE_URL at localhost
go run ./cmd/ofertas-scraper-v2 seed
go run ./cmd/ofertas-scraper-v2 coletar tomate
go run ./cmd/ofertas-scraper-v2 serve
# from repo root
docker compose up --build ofertas-scraper-v2
docker compose run --rm ofertas-scraper-v2 seed
docker compose run --rm ofertas-scraper-v2 coletar tomate
# health: curl http://localhost:8092/health
```

## Tests

```bash
go test ./...
```

No live supermarket HTTP in unit tests — parsers use fixtures; HTTP client uses `httptest`.
