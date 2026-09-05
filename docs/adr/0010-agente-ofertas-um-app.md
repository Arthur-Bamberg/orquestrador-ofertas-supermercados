# One Agente app; catalog assistant is a surface, not a fourth deployable

Interpretation of a Lista against Ofertas lives in `apps/agente-ofertas`, not in the WhatsApp gateway (ADR 0007) and not in `ofertas-api` (catalog CRUD + pipeline). The catalog assistant (MCP) is the same Agente without WhatsApp — one process, two mouths. Rejected: Notion Nível 3 worker + MCP as two services, in-process matching inside the gateway, and a shopping-list session that would contradict Lista-per-Mensagem.
