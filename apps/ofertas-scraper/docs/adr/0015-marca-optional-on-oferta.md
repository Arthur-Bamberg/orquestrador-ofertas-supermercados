# Marca is optional on Oferta

An Oferta always references `produtoId` and `mercadoId`; `marcaId` is optional when the flyer has no brand. Queries that filter by Marca simply omit Ofertas without one. Rejected for the MVP: requiring Marca (drops unbranded goods) and a canonical “Sem marca” sentinel (hides absence behind a fake identity).
