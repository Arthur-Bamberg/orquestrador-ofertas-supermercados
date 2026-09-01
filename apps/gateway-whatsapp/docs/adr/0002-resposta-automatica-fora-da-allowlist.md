# Resposta automática fura a allowlist

A Resposta automática (`WHATSAPP_AUTO_NOME` / `WHATSAPP_AUTO_TEXTO`) envia na Conversa cujo nome casa, mesmo fora da allowlist. `POST /envios` e o Ack de ofertas continuam só na lista. Directa casa `PushName`; grupo casa o assunto. Uma saída por entrada: o nome casa → Resposta automática, senão Ack se permitido. Rejected: reusar o Ack (texto único) e exigir JID na `WHATSAPP_ALLOWLIST`.
