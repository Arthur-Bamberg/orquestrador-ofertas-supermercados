# Documento identity is Fonte + filename + discovery day

A Documento is uniquely identified by its Fonte, PDF filename, and the discovery day in America/Sao_Paulo. The same filename on different days is a new Documento (encartes reuse names); the same triple on the same day is idempotent so re-running the morning job does not duplicate work for `concluido` / `parcial` Documentos.
