# Redis holds current Ofertas/Falhas; Artefatos keep each attempt

Superseded for the store by scraper ADR 0038 (PostgreSQL `SaveAll`); the split “estado atual vs histórico em Artefato” still holds.

On same-day reprocess of a Documento (`falhou` or orphan `processando`), Redis `SaveAll` for Ofertas and Falhas de Extração replaces the previous set for that Documento so the persisted state always matches the latest job outcome. Artefatos, however, retain each processing attempt (PDF, images, raw Extrator response, validated result) without deleting earlier attempts, so prompt and pipeline tuning can compare runs. Rejected alternative: versioning Ofertas/Falhas in Redis — useful for queries, but duplicates the Artefato role and complicates “current price” reads for the MVP.
