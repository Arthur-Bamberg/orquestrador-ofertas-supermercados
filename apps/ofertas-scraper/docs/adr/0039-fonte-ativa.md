# Fonte pode ser ativada ou desativada

Fonte carries `ativa` (default true). The daily job skips inactive Fontes (no HTTP discover, no Documentos). On-demand `discover` rejects an inactive Fonte with `ErrFonteInativa`. Inactive Fontes remain in the catalog for history and admin; delete rules are unchanged. Rejected: deleting a Fonte to pause collection (loses config and blocks FK history); a separate “paused” entity; filtering inactive only in the CLI without persisting the flag.
