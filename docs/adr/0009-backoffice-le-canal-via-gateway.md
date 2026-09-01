# Backoffice lê o canal WhatsApp via HTTP do gateway

The backoffice lists Conversas and Mensagens by calling `gateway-whatsapp` (`VITE_GATEWAY_BASE`, default `:8090`), not by querying schema `whatsapp` from `ofertas-api`. Channel ownership stays in the gateway (ADR 0007). Catalog CRUD stays on `ofertas-api`. GETs of the rastro are unauthenticated like the local catalog MVP; `POST /envios` from the SPA still uses Bearer `GATEWAY_TOKEN` (`VITE_GATEWAY_TOKEN` in the Vite bundle). Rejected: folding the channel into `ofertas-api`, and a nginx BFF that would hide the send token.
