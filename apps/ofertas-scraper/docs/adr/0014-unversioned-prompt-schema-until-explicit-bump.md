# Prompt and schema stay unversioned until an explicit version bump

Until the first running MVP ships and the user explicitly asks for a new Extrator contract version, prompt and schema files use unversioned names (`prompts/extrator.txt`, `schemas/oferta.json`, `schemas/extracao.json`). Do not introduce `v1`/`v2` suffixes prophylactically; version only when a deliberate, incompatible contract change is requested.
