# AGENTS.md — ofertas-scraper

Instructions for coding agents working on **this app** inside the monorepo. Workspace rules (Go modules, compose, hooks, Postgres compartilhado): [`../../AGENTS.md`](../../AGENTS.md). Domain language: [`CONTEXT.md`](./CONTEXT.md). App ADRs: [`docs/adr/`](./docs/adr/). Prefer glossary terms (`Oferta`, `Fonte`, `Documento`, `Extrator`, …) over synonyms.

Module path: `github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper`

## What this system does

Daily job (08:00 America/Sao_Paulo) that:

1. Loads **Fontes** from PostgreSQL
2. GETs each Fonte, discovers `.pdf` names, applies optional per-Fonte filter
3. Creates **Documentos** (identity: Fonte + filename + discovery day)
4. Downloads PDF → rasterizes pages → downscales images → sends to **Extrator**
5. Validates results in **domain** → persists **Ofertas** and **Falhas de Extração**
6. Always stores **Artefatos** for debug

Processing is **sequential** in the MVP (Fonte by Fonte, Documento by Documento).

## Stack

| Concern | Choice |
|---------|--------|
| Language | Go (workspace module) |
| Persistence | PostgreSQL — shared instance with other monorepo apps (`DATABASE_URL`) |
| Local Postgres | Root `docker-compose.yml` |
| Extrator impl | Gemini and/or Cursor (same port; ADR 0023, 0034) |
| Schedule | External cron/systemd timer; binary is a one-shot CLI |
| Timezone | `America/Sao_Paulo` |

## App layout (Clean Architecture)

```
cmd/ofertas-scraper/          # main / DI wiring
internal/
  domain/                     # entities + ports (interfaces)
  application/                # use cases / job flows
  presentation/               # CLI entry only (calls application)
  infra/                      # Gemini, HTTP Fonte, PDF→image, local Artefatos
schemas/oferta.json           # Extrator candidate / Oferta contract
schemas/extracao.json         # Extrator root response (list of candidates)
prompts/extrator.txt          # stable system prompt (bump only when explicitly versioning)
seed/fontes.json
.env.example                  # versioned; copy to .env (gitignored) per machine
CONTEXT.md
docs/adr/
```

Monorepo root owns: `go.work`, `docker-compose.yml`, `.githooks/`, workspace `docs/adr/`.

### Dependency rule

- `presentation` → `application` → `domain`
- `infra` implements Extrator, Fonte HTTP, raster, Artefatos (interfaces in `domain`)
- Catalog ports are implemented in `modules/ofertas-store`; `presentation` wires `store.New*Repo` (workspace DRY / ADR 0038)
- `domain` never imports `infra`, `application`, or `presentation`
- `application` depends on `domain` interfaces only
- Do not import this app’s `internal/` from other apps — extract to `modules/` when sharing (workspace DRY rule)

### Ports (interfaces in `domain`)

Required ports (names may vary; responsibilities must not):

- **MercadoRepository** — list/save Mercados
- **FonteRepository** — list/save Fontes (each Fonte references a Mercado)
- **ProdutoRepository** — list/save Produtos (catalog identity + categorias)
- **MarcaRepository** — list/save Marcas
- **DocumentoRepository** — track Documento lifecycle
- **OfertaRepository** — persist Ofertas linked to Documento (and Produto + Marca + Mercado); listar Documentos por Produto (ADR 0026 / 0038)
- **FalhaExtracaoRepository** — persist Falhas de Extração linked to Documento
- **Extrator** — images in → candidate Ofertas (raw) out
- **ArtefatoStore** — save/load Artefatos (local now; bucket later behind same interface)
- **FonteClient** — HTTP GET + PDF name discovery
- **Rasterizer** — PDF bytes → downscaled page images
- **FilenameDateParser** — parse of vigência (`dataInicio` / `dataExpiracao`) from Documento filename (ADR 0028); DocumentoRepository.EarliestDia for primeira descoberta

## Documento lifecycle

Persisted states: `processando` → `concluido` | `parcial` | `falhou` (`descoberto` is not a persisted state)

| State | Meaning |
|-------|---------|
| `processando` | Download / raster / Extrator in progress (entered at start of treatment) |
| `concluido` | ≥1 Oferta saved, zero Falhas de Extração |
| `parcial` | ≥1 Oferta saved and ≥1 Falha de Extração |
| `falhou` | Hard failure **or** zero Ofertas persistidas (Extrator com `ofertas` vazio, ou só Falhas de Extração) |

Same-day re-run: skip `concluido` and `parcial`; retry `falhou` and orphan `processando`.

## Extrator (Gemini / Cursor adapters)

- System prompt: `prompts/extrator.txt` — must spell out glossary definitions for Produto (sem marca), Marca, and Categoria (taxonômia, não tipo vendável) so extraction stays assertive (ADR 0012)
- Output schema: `schemas/extracao.json` → items conform to `schemas/oferta.json`
- Keep prompt/schema **unversioned** until an explicit version bump is requested (ADR 0014)
- Gemini: cache the **stable prompt** (and schema binding) via context cache when the API allows; **do not** cache Documento images
- Cursor: local Agent SDK via embedded Python bridge (`cursor-sdk`); prompt+schema in the user turn (no native response schema)
- Both adapters: **one API/vision turn per page image**, then merge candidates (ADR 0031) — use case still calls `Extract(images)` once
- Adapter lives in `infra`; use case only sees `Extrator`
- Domain validates every candidate Oferta after extraction (do not trust the model alone)
- **Recall / prompt iteration:** compare Extrator output to Artefato page images via opt-in live tests — see [`docs/extrator-live-recall.md`](./docs/extrator-live-recall.md)

### Oferta rules agents must respect

- Extrator candidates include `produto`, optional `marca`, `categorias[]`, plus `valor` / `quantidades[]` / `medida` / `dataInicio` / `dataExpiracao` / optional `promocao` / optional `comparativo` (see ADR 0010, 0030, 0035); persisted Oferta stores `produtoId`, `mercadoId`, optional `marcaId`, vigência + `origemDataInicio` / `origemDataExpiracao` after match-or-create (ADR 0015, 0028)
- `quantidades` is a non-empty array of sizes sharing one price and one `medida`; domain dedups and sorts ascending; same-price discrete lists on the flyer → one Oferta; readers accept legacy singular `quantidade` as `[n]` (ADR 0030)
- `medida` is only `g` | `ml` | `unidade`
- Extrator must normalize **kg → 1000 g** and **L → 1000 ml** (adjust each value in `quantidades`) before output; domain does **not** convert — any other `medida` is a Falha de Extração (see ADR 0004; hybrid domain safety-net deferred)
- `dataInicio` / `dataExpiracao` = vigência no encarte; both required in Extrator contract; cascades always on (ADR 0028): Extrator wins when present; missing início → distinct start in filename → primeira descoberta na Fonte; missing fim → filename end, else Falha; past/future dates OK (ADR 0016); `dataInicio` ≤ `dataExpiracao`
- `promocao` is optional: `valorPromocional` plus channel (cartão XOR clube) and/or quantity mechanic (leve/pague XOR quantidadePromocao); channel+mechanic may compose when they share the same price (ADR 0032; clube shape in ADR 0029)
- `comparativo` is optional: `{ quantidade, valor }` pack-fraction / “sai por nesta embalagem” badge; Medida inherited from Oferta; not Promoção and not a second Oferta (ADR 0035)
- Each Extrator tentativa persists **Uso do Extrator** (prompt/cache/output tokens + Artefato path) to Postgres and `uso-extrator.json` (ADR 0033)
- Domain match-or-create for Produto/Marca uses normalized exact label match only (ADR 0011); no fuzzy matching in the MVP

## Artefatos

For each processing attempt, persist **best-effort** what the pipeline produced (ADR 0027):

1. Original PDF (if download succeeded)
2. Images sent to Extrator (if raster succeeded)
3. Raw Extrator response (if Extract returned)
4. Uso do Extrator (`uso-extrator.json`) when usage was reported
5. Validated result (Ofertas + Falhas de Extração) when validation ran

Do not write empty placeholders for steps that never ran. Store via `ArtefatoStore` only — never write files ad hoc from use cases.

## Local environment

- From **monorepo root**: copy `.env.example` → `.env`, then `docker compose up` starts Postgres
- App env: `apps/ofertas-scraper/.env` (gitignored) + `.env.example` (versioned)
- Paths in `.env` (`SEED_PATH`, `ARTEFATO_ROOT`, schemas, prompts) are **relative to this app directory** — run the CLI from here
- Artefatos default: `./.data/artefatos` (gitignored; not ported)
- Git hooks: install once from monorepo root (`../../scripts/install-git-hooks.sh`) — workspace ADR 0003 (supersedes app ADR 0024 for hook location)

### Suggested env vars

```
DATABASE_URL=postgres://ofertas:ofertas@localhost:5432/ofertas?sslmode=disable
EXTRATOR_PROVIDER=auto
EXTRATOR_STUB=1
GEMINI_API_KEY=
GEMINI_MODEL=gemini-3-flash-preview
CURSOR_API_KEY=
CURSOR_MODEL=composer-2.5
EXTRATOR_PROMPT_PATH=./prompts/extrator.txt
EXTRACAO_SCHEMA_PATH=./schemas/extracao.json
OFERTA_SCHEMA_PATH=./schemas/oferta.json
SEED_PATH=./seed/fontes.json
ARTEFATO_ROOT=./.data/artefatos
RASTER_MAX_EDGE_PX=1280
RASTER_JPEG_QUALITY=80
TZ=America/Sao_Paulo
```

With `EXTRATOR_PROVIDER=auto` (default), Gemini is primary and Cursor is failover on rate-limit (ADR 0037).

CLI (from this directory):

```bash
go run ./cmd/ofertas-scraper seed
go run ./cmd/ofertas-scraper run
```

Rasterizer needs `pdftoppm` (poppler-utils) on PATH.

## Agent do's and don'ts

**Do**

- Use glossary terms from `CONTEXT.md`
- Follow workspace rules in root `AGENTS.md` (modules, shared Postgres, DRY extract)
- Add/change persistence and Extrator only behind `domain` interfaces
- Keep job orchestration in `application`
- Keep prompt + schema in sync; only introduce versioned filenames when explicitly asked to bump the Extrator contract (ADR 0014)
- Follow workspace TDD in [`../../AGENTS.md`](../../AGENTS.md): one observable behavior — failing test → code that passes → refactor the test → refactor the code
- Prefer small, sequential changes with tests around domain validation
- After **any** prompt or Extrator schema change, run live recall and **validate manually** against page images (`docs/extrator-live-recall.md`) — automated floors alone are not enough

**Don't**

- Call Gemini or Postgres from `domain` or `application` directly
- Put ports in `presentation` (CLI only)
- Send raw PDF bytes to the Extrator (images only)
- Hardcode Fonte URLs (they live in the catalog)
- Parallelize Fontes/Documentos in the MVP without an explicit decision
- Commit secrets (`.env`, API keys)
- Skip pre-commit with `--no-verify` unless the user explicitly asks

## Pipeline reference

```
cron 08:00 America/Sao_Paulo
  → CLI (presentation)
    → RunDailyJob (application)
      → FonteRepository.List()
      → for each Fonte (sequential):
          FonteClient.DiscoverPDFs(+ filter)
          → for each new Documento:
              download PDF
              ArtefatoStore.Save(pdf)
              Rasterizer → images (max edge 1280px)
              ArtefatoStore.Save(images)
              Extrator.Extract(images)
              ArtefatoStore.Save(raw)
              domain.Validate → Ofertas | Falhas
              persist + ArtefatoStore.Save(validated)
              update Documento state
```
