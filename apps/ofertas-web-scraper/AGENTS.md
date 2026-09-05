# AGENTS.md — ofertas-web-scraper

Coleta de preços nos sites dos Mercados. Glossário: [`../ofertas-scraper/CONTEXT.md`](../ofertas-scraper/CONTEXT.md) (Coleta, Oferta, Indicação Promocional). Workspace: [`../../AGENTS.md`](../../AGENTS.md). ADRs: [`docs/adr/`](./docs/adr/), workspace [`../../docs/adr/0011-ofertas-web-scraper-coleta.md`](../../docs/adr/0011-ofertas-web-scraper-coleta.md).

Module path: `github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper`

## What this system does

One-shot CLI. Unit of work is **one Produto** (`coletar <produtoId>`):

1. For each Vitrine (Fort, Carrefour, Asun, Rissul) search the Produto name
2. Keep cards whose label contains the Produto `nomeNorm`
3. Persist a **Coleta** (Produto + Mercado + day) and matching **Ofertas**
4. Vigência = that day; Indicação Promocional from the card (encarte path always true — not this app)

`run` iterates every Produto with the same unit.

## Stack

| Concern | Choice |
|---------|--------|
| Language | Go |
| Persistence | `modules/ofertas-store` (shared Postgres) |
| Schedule | External; `docker compose run --rm ofertas-web-scraper coletar <id>` |

## Layout

```
cmd/ofertas-web-scraper/
internal/
  domain/
  application/
  presentation/
  infra/vitrine/{fort,carrefour,asun,rissul}
seed/mercados.json
```

## Local

```bash
# from this directory, with root .env DATABASE_URL at localhost
go run ./cmd/ofertas-web-scraper seed
go run ./cmd/ofertas-web-scraper coletar <produtoId>
go run ./cmd/ofertas-web-scraper run
# from repo root
docker compose run --rm ofertas-web-scraper seed
docker compose run --rm ofertas-web-scraper coletar <produtoId>
```

## Tests

```bash
go test ./...
```

No live supermarket HTTP in unit tests — parsers use fixtures; HTTP client uses `httptest`.
