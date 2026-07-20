# Go workspace with one module per app

This repo is a Go workspace (`go.work`) so multiple apps can share a single clone while keeping independent `go.mod` files. Each deployable lives under `apps/<name>` with module path `github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/<name>`. Shared libraries are extracted into `modules/<name>` only when a second consumer needs the same code (DRY — never copy between apps). App-private code stays in `apps/*/internal`.
