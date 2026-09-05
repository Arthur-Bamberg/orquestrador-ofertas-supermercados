# AGENTS.md — orquestrador-ofertas-supermercados

Instructions for coding agents on this **monorepo / Go workspace**. Domain language: scraper [`apps/ofertas-scraper/CONTEXT.md`](./apps/ofertas-scraper/CONTEXT.md); WhatsApp channel [`apps/gateway-whatsapp/CONTEXT.md`](./apps/gateway-whatsapp/CONTEXT.md); Agente [`apps/agente-ofertas/CONTEXT.md`](./apps/agente-ofertas/CONTEXT.md); map [`CONTEXT-MAP.md`](./CONTEXT-MAP.md). Workspace ADRs in [`docs/adr/`](./docs/adr/); app ADRs under each `apps/<name>/docs/adr/`.

## What this repository is

Workspace for supermarket-offers apps. It is **not** itself an application binary.

| Status | App | Role |
|--------|-----|------|
| **Current** | `ofertas-scraper` | Daily job: Fontes → Documentos → Extrator → Ofertas |
| **Current** | `ofertas-api` | Go HTTP API over shared Postgres (CRUD + pipeline ops; ADR 0005 / 0006) |
| **Current** | `ofertas-backoffice` | Vite/React SPA backoffice (filters + screens; ADR 0005) |
| **Current** | `gateway-whatsapp` | WhatsApp channel gateway (whatsmeow; ADR 0007) |
| **Current** | `agente-ofertas` | Interprets Lista, reads Oferta, replies via Canal; catalog assistant is a surface (ADR 0010) |

Create an app module only when implementing it — do not scaffold empty `apps/` directories.

## Stack (workspace)

| Concern | Choice |
|---------|--------|
| Language | Go (`go.work`, Go 1.24+) |
| Layout | `apps/<name>` per deployable; `modules/<name>` for shared libs (lazy) |
| Local infra | Root [`docker-compose.yml`](./docker-compose.yml) — Postgres + apps (ADR 0008) |
| Deploy images | `apps/<name>/Dockerfile` |
| Shared data | One PostgreSQL instance for all apps (ADR 0006) |
| Schedule / TZ | Per app (scraper: external cron or `docker compose run`, `America/Sao_Paulo`) |

## Repository layout

```
go.work
docker-compose.yml              # local stack (Postgres + apps)
AGENTS.md                       # this file (workspace)
docs/adr/                       # workspace / platform decisions
apps/
  ofertas-scraper/              # first app (own go.mod, AGENTS.md, CONTEXT.md, ADRs)
  ofertas-api/
  ofertas-backoffice/
  gateway-whatsapp/             # WhatsApp channel (ADR 0007)
  agente-ofertas/               # Lista → Oferta → Resposta (ADR 0010)
modules/                        # shared Go modules — create only when extracting
.githooks/                      # pre-commit → test every workspace module
scripts/install-git-hooks.sh
```

### Module paths

```
github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/<app>
github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/<lib>
```

Register new modules in root `go.work` (`use ./apps/...` or `./modules/...`).

### Dependency / sharing rules

- Code under `apps/<name>/internal` is **private to that module** — other apps must not import it.
- **DRY across apps:** never copy packages between apps. When a second app needs the same code, **extract** it to `modules/<name>` and depend on that module.
- Prefer extracting after the second consumer is real — do not invent empty `modules/` “just in case”.
- `presentation` → `application` → `domain` (and `infra` implements domain ports) inside each app unless an ADR says otherwise.

## Shared PostgreSQL

- One Postgres for all apps (local via compose; cloud via any `DATABASE_URL`).
- **Domain tables** (Oferta, Documento, Produto, …): shared contract in `modules/ofertas-store` (scraper ADR 0038). Any app may read/write through that module.
- **Operational / channel state** (e.g. WhatsApp send tracking): separate tables — do not stuff into Oferta/Documento (workspace ADR 0006).
- Env: scraper/api/backoffice keep `.env` per app for host-side `go run` / `npm`. `gateway-whatsapp` and Compose read the **root** `.env`. Point host `DATABASE_URL` at `localhost`; containers use hostname `postgres` (ADR 0008).

## Local environment

```bash
cp .env.example .env              # senha local do Postgres + canal + extrator
docker compose up --build         # from repo root — Postgres, API :8080, backoffice :5173, gateway :8090, agente :8091
docker compose run --rm ofertas-scraper seed
docker compose run --rm ofertas-scraper run
./scripts/install-git-hooks.sh    # once per clone
cd apps/ofertas-scraper && cp .env.example .env   # only if you `go run` the CLI on the host
# host scraper: paths in apps/ofertas-scraper/.env are relative to that cwd
# host gateway: go run ./apps/gateway-whatsapp/cmd/gateway-whatsapp   # optional; Compose is the default
```

### Git hooks

```bash
./scripts/install-git-hooks.sh
```

Pre-commit runs `go test ./...` in **every** module in `go.work`, then `npm test` in `apps/ofertas-backoffice`. Do not use `--no-verify` unless explicitly required.

## TDD (required)

Grow production behavior with TDD. One **observable behavior** per cycle, through the public API of the package, type, or function — not an empty constructor test.

1. Write the smallest **failing** test for the next behavior (RED)
2. Write the smallest code that makes it pass (GREEN)
3. Refactor the **test** (glossary names from `CONTEXT.md`, duplication, clarity)
4. Refactor **production** code (tests stay green)

Do **not** write all tests for a unit before any production code (horizontal slice).

**Existing code:** characterization tests (test after the fact) lock current behavior; a failure is a bug to fix. Day-to-day work on new behavior uses the loop above.

**Unit tests here:** ephemeral Postgres via `modules/ofertas-store/storetest` (embedded-postgres), no live Extrator, no `pdftoppm`. In-memory ports count as unit. Live Extrator stays opt-in (`LIVE_EXTRATOR=1`).

## Git workflow (agents)

Default for this repo: work on **`main`**. Do **not** create a feature branch, open a pull request, or invent a PR flow unless the user **explicitly** asks for a branch and/or PR.

- “Commit” / “push” alone → commit and push on the current branch (usually `main`); no branch, no PR.
- Skills, templates, or habits that prefer branch+PR **do not override** this — only an explicit user request does.

## Agent do's and don'ts

**Do**

- Read the app’s `AGENTS.md` + `CONTEXT.md` when working inside that app
- Add new apps as `apps/<name>` with own `go.mod` and register in `go.work`
- Extract shared code to `modules/` instead of duplicating
- Keep secrets out of git (`.env` per app, never commit)
- Install hooks after cloning
- Commit/push on `main` when asked, unless the user asked for a branch/PR

**Don't**

- Scaffold planned apps before implementation
- Import another app’s `internal/` packages
- Copy-paste shared logic across apps
- Put domain operational hacks into shared Oferta rows without an ADR
- Commit `.env`, API keys, or `.data/` artefacts
- Skip pre-commit with `--no-verify` unless the user explicitly asks
- Create a branch or PR “by default” or because a skill suggests it — only when the user asks
- Skip the TDD loop for new behavior (all tests first, or production before a failing test)

## App-specific docs

| App | Agents | Domain |
|-----|--------|--------|
| ofertas-scraper | [`apps/ofertas-scraper/AGENTS.md`](./apps/ofertas-scraper/AGENTS.md) | [`apps/ofertas-scraper/CONTEXT.md`](./apps/ofertas-scraper/CONTEXT.md) |
| ofertas-api | [`apps/ofertas-api/AGENTS.md`](./apps/ofertas-api/AGENTS.md) | Same glossary as scraper (`CONTEXT.md` above; ADR 0005) |
| ofertas-backoffice | [`apps/ofertas-backoffice/AGENTS.md`](./apps/ofertas-backoffice/AGENTS.md) | Same glossary as scraper (`CONTEXT.md` above; ADR 0005) |
| gateway-whatsapp | [`apps/gateway-whatsapp/AGENTS.md`](./apps/gateway-whatsapp/AGENTS.md) | [`apps/gateway-whatsapp/CONTEXT.md`](./apps/gateway-whatsapp/CONTEXT.md) |
| agente-ofertas | [`apps/agente-ofertas/AGENTS.md`](./apps/agente-ofertas/AGENTS.md) | [`apps/agente-ofertas/CONTEXT.md`](./apps/agente-ofertas/CONTEXT.md) |
