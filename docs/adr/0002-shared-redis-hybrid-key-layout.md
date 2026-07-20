# Shared Redis with hybrid key layout

All apps use the same Redis/Upstash instance for local and cloud. Domain keys (Mercado, Fonte, Documento, Oferta, Produto, Marca, …) follow the scraper contract (see `apps/ofertas-scraper` ADR 0019) and any app may read or write them. Operational state that is not domain language (e.g. WhatsApp delivery tracking) lives under an app-specific key prefix so channel concerns do not pollute Oferta/Documento records. Extract a shared Go client into `modules/` when two apps need the same access code.
