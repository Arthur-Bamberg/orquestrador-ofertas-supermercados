# AGENTS.md - ofertas-api

Go HTTP API for supermarket offers administration and pipeline operations.

- Module path: `github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-api`
- Persistence must go through `modules/ofertas-store`; do not import `apps/ofertas-scraper/internal`.
- Domain tables are shared with the scraper (ADR 0006). API operational state is `operacao_pipeline` in the same database.
- No authentication is implemented in this app.
- Keep the scraper CLI as the execution boundary for pipeline operations (`go run` on the host, or `SCRAPER_BIN` inside the API image — ADR 0008).
- `POST /api/ops/testar` (`mercadoId`, `termo` when Fonte `tipo=site`) scans encarte via discover or scrapes via `COLETA_URL` / `GATEWAY_TOKEN` (ADR 0015). Does not call the Extrator.
- Follow workspace TDD in [`../../AGENTS.md`](../../AGENTS.md): one observable behavior — failing test → code that passes → refactor the test → refactor the code.
