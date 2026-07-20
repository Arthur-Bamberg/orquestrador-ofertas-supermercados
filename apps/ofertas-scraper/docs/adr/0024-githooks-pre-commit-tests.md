# Versioned Git hooks run `go test ./...` before commit

> **Superseded for hook location** by monorepo [`docs/adr/0003-githooks-workspace-tests.md`](../../../docs/adr/0003-githooks-workspace-tests.md): hooks live at the workspace root and test every `go.work` module. The intent (versioned pre-commit tests, no Husky) still holds.

There is no Husky equivalent in the Go toolchain. The original app-local approach used a versioned `.githooks/` directory and `git config core.hooksPath .githooks` (via `scripts/install-git-hooks.sh`) so every commit runs `go test ./...` with zero Node/Python/Lefthook dependencies. Hooks are opt-in per clone after install; CI remains the backstop if someone skips hooks.
