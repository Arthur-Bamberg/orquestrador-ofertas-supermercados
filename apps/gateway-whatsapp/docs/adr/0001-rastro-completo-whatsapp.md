# Rastro completo do canal no schema whatsapp

O gateway persiste toda Mensagem viva e do history sync (direta, grupo, Status, FromMe, reacção, mídia) no schema `whatsapp`. Contato/Conversa guardam PN e LID para não duplicar a thread. Allowlist como portão de Ack/`POST /envios` do produto: superseded by workspace [ADR 0019](../../../docs/adr/0019-aceite-no-contato-em-vez-de-allowlist.md) (Allowlist fica só no compositor do Operador). Rejected: silêncio no INSERT (feature-loop antigo D4), ignorar `events.HistorySync`, e presença/digitando/chamadas (não são Mensagem).
