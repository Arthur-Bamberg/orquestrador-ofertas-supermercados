# Coleta identity is termo + Mercado + dia; persist every complete card

Oferta de vitrine still comes from **Coleta**, not Documento. The Coleta key is the Item **termo** + Mercado + civil day (`America/Sao_Paulo`), because the search is the person's wording and one busca yields many Produtos (related cards included). Every complete card is persisted (Produto/Marca match-or-create); Filtragem only affects Resposta. Rejeitado: identity Produto+Mercado+dia (ADR 0011) and dropping related cards before persist.

Status: accepted. Same-day reuse / replace amended by [ADR 0014](./0014-coleta-reusa-mesmo-dia.md). Supersedes the Coleta identity and “only matching cards” parts of [ADR 0011](./0011-ofertas-web-scraper-coleta.md).

