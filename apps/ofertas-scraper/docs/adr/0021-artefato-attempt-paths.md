# Artefato paths version each processing attempt by timestamp

Local ArtefatoStore writes under `{ARTEFATO_ROOT}/{fonteId}/{dia}/{filename}/{tentativa}/` with `original.pdf`, `images/page-NNN.jpg`, `extrator-raw.json`, and `validated.json` when those steps ran (best-effort — ADR 0027). `tentativa` is a UTC timestamp `YYYYMMDDTHHMMSS` taken when processing starts, so same-day reprocess appends a new directory and leaves prior attempts intact (CONTEXT Artefato; ADR 0017). Sequential MVP makes timestamp collisions negligible; numeric attempt counters were rejected as extra state for no gain.
