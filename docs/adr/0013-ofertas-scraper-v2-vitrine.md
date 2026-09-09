# ofertas-scraper-v2 is the vitrine Coleta app

Site prices enter the catalog through `ofertas-scraper-v2`: HTTP `POST /coletas` (Agente da Lista) plus CLI `seed` / `coletar`. This recorte is Fort only; loja = the site default (Balneário Camboriú display `115`, Sense store id `1585`), not a hardcoded Canoas store. Sense search uses page size 12 and header `X-Osuper-Search-Token` from the Fort home HTML. The app is long-running in Compose (`:8092`). `ofertas-web-scraper` is removed — Carrefour/Asun/Rissul have no site Coleta until their v2 adapters. Encarte PDF stays in `ofertas-scraper`. Shopfully does not enter this app. Rejeitado: estender `ofertas-web-scraper` and merging the Shopfully `ofertas-scraper-v2` branch.

Status: accepted. Supersedes [ADR 0011](./0011-ofertas-web-scraper-coleta.md) as the home of vitrine Coleta.
