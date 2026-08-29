# Versioned Git hooks run workspace tests before commit

There is no Husky equivalent in the Go toolchain. The monorepo versions `.githooks/` at the repo root and installs via `scripts/install-git-hooks.sh` (`core.hooksPath=.githooks`). Pre-commit runs `go test ./...` in every module listed by `go.work`, then `npm test` in `apps/ofertas-backoffice`. This supersedes the app-local hook approach from `apps/ofertas-scraper` ADR 0024 for this repository.
