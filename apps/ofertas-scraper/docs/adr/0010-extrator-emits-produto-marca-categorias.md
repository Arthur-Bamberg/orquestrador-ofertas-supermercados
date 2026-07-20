# Extrator emits produto, optional marca, and categorias as labels

Each Extrator candidate includes `produto` as a string, optional `marca`, and `categorias` as a string array, alongside price fields (valor, quantidade, medida, dataExpiracao, promocao). Domain match-or-create resolves `produtoId` and, when marca is present, `marcaId`; `mercadoId` comes from the Fonte, never from the Extrator. Rejected: a single free-text `nome` for the Extrator to split, or deferring categorias entirely to later curation.
