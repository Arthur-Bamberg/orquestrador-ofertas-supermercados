# Uso do Extrator: token usage per Documento tentativa

Superseded for the store by scraper ADR 0038: Uso lives in `uso_extrator`; Artefato `uso-extrator.json` is unchanged. Live Extrator tests still write the file only (no catalog).

Each Extrator attempt records token usage (prompt, cache, output) with the Artefato path for that tentativa, plus which adapter (`provider`: `gemini` | `cursor` | `stub`) and `model` completed the call (ADR 0037). Redis key `uso-extrator:{documentoId}:{tentativa}` holds the JSON; SET `uso-extrator:documento:{documentoId}` indexes tentativas. The same payload is written as `uso-extrator.json` under the Artefato directory. Live Extrator tests write the file only (no Redis). Rejected: embedding usage on Documento (loses history on reprocess) and storing only dollars (token counts are stable; pricing is external).
