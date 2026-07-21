# AGENTS.md - ofertas-api

Go HTTP API for supermarket offers administration and pipeline operations.

- Module path: `github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-api`
- Persistence must go through `modules/ofertas-store`; do not import `apps/ofertas-scraper/internal`.
- Domain Redis keys are shared with the scraper. API operational state uses `ofertas-api:ops:*`.
- No authentication is implemented in this app.
- Keep the scraper CLI as the execution boundary for pipeline operations.
