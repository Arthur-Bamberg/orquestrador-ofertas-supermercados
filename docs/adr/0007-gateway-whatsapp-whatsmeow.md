# Gateway WhatsApp with whatsmeow and a separate channel schema

The WhatsApp deployable is `apps/gateway-whatsapp`: a always-on Go process using whatsmeow (QR pairing, sqlite device store under `.data/`), not Cloud API webhooks and not Cloud Run scale-to-zero. Channel state (Contato, Conversa, Mensagem) lives in PostgreSQL schema `whatsapp` with its own `whatsapp.schema_migrations`, not in `modules/ofertas-store`. Media bytes stay on disk. Rejected: folding the channel into `ofertas-api`, in-process oferta matching, and NATS until a second process needs it.

Status: superseded (transport and always-on deploy) by [0020](./0020-gateway-whatsapp-cloud-api.md). Schema `whatsapp` still stands.
