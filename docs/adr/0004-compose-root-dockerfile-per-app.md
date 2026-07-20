# Docker Compose at repo root; Dockerfile per app at deploy time

Local multi-app stacks are defined in the root `docker-compose.yml` so one `docker compose up` can run shared infrastructure (and later app services) together. Production packaging is a `Dockerfile` inside each `apps/<name>` when that app is hosted — not invented in the initial port. Compose stays the local orchestration surface; images stay per-app.
