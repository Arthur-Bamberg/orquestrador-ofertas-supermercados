# Rastro completo do canal no schema whatsapp

O gateway persiste toda Mensagem viva e do history sync (direta, grupo, Status, FromMe, reacção, mídia) no schema `whatsapp`; a allowlist só controla ack e `POST /envios`. Contato/Conversa guardam PN e LID para não duplicar a thread. Rejected: silêncio no INSERT (feature-loop antigo D4), ignorar `events.HistorySync`, e presença/digitando/chamadas (não são Mensagem).
