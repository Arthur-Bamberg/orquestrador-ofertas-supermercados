# AGENTS.md — ofertas-scraper-v2

Shopfully image scraper. Workspace: [`../../AGENTS.md`](../../AGENTS.md). Domain: [`CONTEXT.md`](./CONTEXT.md). App ADRs: [`docs/adr/`](./docs/adr/).

Module path: `github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2`

## What this system does

One-shot CLI that:

1. GETs the Shopfully Canoas supermercados listing
2. Parses Encarte cards (Mercado + flyerId + viewer path)
3. Keeps only Mercados from `seed/mercados.json` (the ones we already have)
4. Opens each viewer → `publicationId` → Zmags page bundles
5. Downloads the highest-resolution JPEG of each Página

Does **not** rasterize PDFs, call the Extrator, or write the catalog.

## Stack

| Concern | Choice |
|---------|--------|
| Language | Go (workspace module) |
| Fonte | `https://www.shopfully.com.br/canoas/supermercados` |
| Images | local `ARTEFATO_ROOT` (`{mercado}/{encarteId}/pagina-NN.jpeg`) |
| Timezone | `America/Sao_Paulo` |

## Layout

```
cmd/ofertas-scraper-v2/
internal/
  domain/
  application/
  presentation/
  infra/shopfully/
  infra/artefato/
seed/mercados.json
```

`presentation` → `application` → `domain`. `infra` implements the Shopfully HTTP port.

## Local

```bash
cd apps/ofertas-scraper-v2
go run ./cmd/ofertas-scraper-v2 discover
go run ./cmd/ofertas-scraper-v2 run
# or from repo root:
docker compose run --rm ofertas-scraper-v2 discover
docker compose run --rm ofertas-scraper-v2 run
```

## Tests

```bash
go test ./...
```

Unit tests use fixtures + `httptest`. Live Shopfully is opt-in via the CLI, not CI.
