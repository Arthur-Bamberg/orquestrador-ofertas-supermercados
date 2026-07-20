# Empty or all-rejected extraction yields Documento falhou

When processing finishes with zero persisted Ofertas — Extrator OK with empty `ofertas`, or every candidate becomes a Falha de Extração — the Documento ends in `falhou`, not `concluido` or `parcial`. Those success states require at least one Oferta; Falhas may still be stored under a `falhou` Documento for debug. Same-day re-runs therefore retry these Documentos. Ops can later alert on chronic empty Fontes without redefining success states.
