# Coleta de vitrine é um app; Oferta de site não é Documento

Preço observado no site do Mercado entra no catálogo como Oferta, mas a origem é **Coleta**, não um Documento-fantasma nem uma Fonte. `ofertas-scraper` continua o job de encarte — o rename do módulo não vale o churn. Vigência da Oferta de Coleta é só o dia da Coleta (`origem` `coleta`). Indicação Promocional é boolean distinto de Promoção: verdadeira em todo encarte; no site sai do cartão. Rejeitado: esticar Documento para HTML, Oferta órfã de site, e misturar o boolean com o objeto Promoção.

Status: superseded by [ADR 0012](./0012-coleta-termo-mercado-dia.md) (Coleta identity and persist-all cards) and [ADR 0013](./0013-ofertas-scraper-v2-vitrine.md) (`ofertas-scraper-v2` replaces `ofertas-web-scraper`).

