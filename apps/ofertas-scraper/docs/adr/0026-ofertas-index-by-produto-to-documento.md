---
status: accepted
---

# Secondary index: Produto → Documentos that hold Ofertas

Superseded for persistence by scraper ADR 0038: lookup is `documento_oferta` JOIN `oferta` (index `oferta.produto_id`), not a Redis SET. The query “Documentos that hold Ofertas of this Produto” still holds.

Price history for a Produto is the set of Ofertas over time; clients filter the current period themselves. Redis already stores Ofertas as one JSON blob per Documento (`ofertas:documento:{documentoId}`, ADR 0017/0019). To find that history without embedding Ofertas on Produto or one-key-per-Oferta, maintain a SET `ofertas:produto:{produtoId}` of `documentoId` values. On each `SaveAll` for a Documento, remove that Documento from index entries of Produtos that dropped out of the new set and add it for Produtos present in the new set. Rejected for MVP: indexing individual Oferta ids (requires addressable `oferta:{id}` keys) and embedding history on the Produto JSON.
