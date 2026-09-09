# Same-day Coleta reuses concluído/parcial; only falhou and orphan processando replace

Coleta identity remains termo + Mercado + day (`America/Sao_Paulo`). A later call of that trio does **not** scrape again when the Coleta is `concluido` or `parcial` — it returns the persisted Ofertas (including a partial page set or a timed-out page set already saved). `falhou` and orphan `processando` retry and then replace the set. In-flight `processando` waits and shares one search. Empty exhausted search is `concluido` with zero Ofertas (saved); cut-off with zero Ofertas is `falhou`. Rejected: always replacing on every call (ADR 0012 wording) and treating truncated search as `falhou` when Ofertas already exist.

Status: accepted. Amends the “next Coleta replaces the set” part of [ADR 0012](./0012-coleta-termo-mercado-dia.md).
