# AGENTS.md — orquestrador-ofertas-supermercados

Instructions for coding agents on this **monorepo / Go workspace**. Domain language for the scraper lives in [`apps/ofertas-scraper/CONTEXT.md`](./apps/ofertas-scraper/CONTEXT.md); workspace ADRs in [`docs/adr/`](./docs/adr/); app ADRs under each `apps/<name>/docs/adr/`.

## What this repository is

Workspace for supermarket-offers apps. It is **not** itself an application binary.

| Status | App | Role |
|--------|-----|------|
| **Current** | `ofertas-scraper` | Daily job: Fontes → Documentos → Extrator → Ofertas |
| **Current** | `ofertas-api` | Go HTTP API over shared Redis (CRUD + pipeline ops; ADR 0005) |
| **Current** | `ofertas-backoffice` | Vite/React SPA backoffice (filters + screens; ADR 0005) |
| **Planned** | `gateway-whatsapp` | WhatsApp channel gateway |
| **Planned** | `agente-ofertas-worker` | Agent/worker over ofertas |
| **Planned** | `mcp-server-ofertas` | MCP server for ofertas |

Create an app module only when implementing it — do not scaffold empty `apps/` directories.

## Stack (workspace)

| Concern | Choice |
|---------|--------|
| Language | Go (`go.work`, Go 1.24+) |
| Layout | `apps/<name>` per deployable; `modules/<name>` for shared libs (lazy) |
| Local infra | Root [`docker-compose.yml`](./docker-compose.yml) |
| Deploy images | `apps/<name>/Dockerfile` when hosting that app (not required at port time) |
| Shared data | One Redis/Upstash instance for all apps (ADR 0002) |
| Schedule / TZ | Per app (scraper: external cron, `America/Sao_Paulo`) |

## Repository layout

```
go.work
docker-compose.yml              # local stack (Redis + SRH today)
AGENTS.md                       # this file (workspace)
docs/adr/                       # workspace / platform decisions
apps/
  ofertas-scraper/              # first app (own go.mod, AGENTS.md, CONTEXT.md, ADRs)
  # gateway-whatsapp/           # planned
  # agente-ofertas-worker/      # planned
  # mcp-server-ofertas/         # planned
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

## Shared Redis

- One Redis for all apps (local via compose SRH; cloud via Upstash).
- **Domain keys** (Oferta, Documento, Produto, …): shared contract from the scraper (see app ADR 0019). Any app may read/write.
- **Operational / channel state** (e.g. WhatsApp send tracking): app-specific key **prefix** — do not stuff into Oferta/Documento (workspace ADR 0002).
- Env: each app has its own `.env` (gitignored) + `.env.example` (versioned). Point `UPSTASH_*` at the same instance when sharing data.

## Local environment

```bash
docker compose up                 # from repo root — Redis + SRH
./scripts/install-git-hooks.sh    # once per clone
cd apps/ofertas-scraper && cp .env.example .env   # if needed
# run scraper from apps/ofertas-scraper (paths in .env are relative to app cwd)
```

### Git hooks

```bash
./scripts/install-git-hooks.sh
```

Pre-commit runs `go test ./...` in **every** module in `go.work`. Do not use `--no-verify` unless explicitly required.

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
- Put domain operational hacks into shared Oferta keys without an ADR
- Commit `.env`, API keys, or `.data/` artefacts
- Skip pre-commit with `--no-verify` unless the user explicitly asks
- Create a branch or PR “by default” or because a skill suggests it — only when the user asks

## App-specific docs

| App | Agents | Domain |
|-----|--------|--------|
| ofertas-scraper | [`apps/ofertas-scraper/AGENTS.md`](./apps/ofertas-scraper/AGENTS.md) | [`apps/ofertas-scraper/CONTEXT.md`](./apps/ofertas-scraper/CONTEXT.md) |
| ofertas-api | [`apps/ofertas-api/AGENTS.md`](./apps/ofertas-api/AGENTS.md) | Same glossary as scraper (`CONTEXT.md` above; ADR 0005) |
| ofertas-backoffice | [`apps/ofertas-backoffice/AGENTS.md`](./apps/ofertas-backoffice/AGENTS.md) | Same glossary as scraper (`CONTEXT.md` above; ADR 0005) |
