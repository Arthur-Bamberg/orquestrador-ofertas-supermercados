# Backoffice can test encarte scan or site scraping per Mercado

Admin needs to exercise collection without the Extrator: scan PDFs on a Fonte `tipo=encarte`, or scrape vitrine on `tipo=site`. `POST /api/ops/testar` with `mercadoId` (and `termo` when site) looks up that Mercado’s Fonte: encarte enqueues the existing `discover` Operação de Pipeline (Documentos, no Extrator); site proxies to `ofertas-scraper-v2` `POST /coletas` with `mercadoId` (Coleta, no IA). Rejected: a new Operação kind for Coleta (scraper-v2 is already HTTP; the API image has no v2 CLI) and always calling `discover` (site Fontes are refused by the scraper).

Status: accepted.
