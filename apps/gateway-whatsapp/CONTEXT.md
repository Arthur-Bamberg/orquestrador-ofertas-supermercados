# Gateway WhatsApp

Canal WhatsApp do workspace de ofertas: recebe e envia Mensagens (texto e Mídia) em Conversas diretas, de grupo ou Status, sem consultar Oferta nem interpretar lista de compras.

## Language

**Contato**:
Pessoa no canal, identificada pelo JID canónico de telefone (`…@s.whatsapp.net`) quando o evento traz PN; o LID (`…@lid`) fica em `jid_lid`.
_Avoid_: usuário, cliente, user, sender (como entidade)

**Conversa**:
Thread do canal: **direta** (1:1), **grupo** (`…@g.us`) ou **status** (`status@broadcast`). A allowlist vale sobre o JID da Conversa só para **Ack** e `POST /envios`; o rastro persiste em todas. A Resposta automática casa o nome, não a lista.
_Avoid_: chat, sala, thread

**Mensagem**:
Unidade persistida de entrada ou saída numa Conversa: corpo de texto, Mídia opcional, id do provedor (idempotência), origem (`vivo` ou `historico`), timestamp do WhatsApp e, na saída, status de envio (`pendente`, `enviado`, `falhou`, `entregue`, `lido`, `reproduzido`). Inclui FromMe, reacção, revogação e indecifrável.
_Avoid_: evento, payload, Envio (como tabela)

**Mídia**:
Anexo da Mensagem — imagem, áudio, vídeo, documento ou figurinha — bytes no disco do gateway quando o download consegue, metadados na Mensagem mesmo se falhar. O gateway não interpreta o conteúdo.
_Avoid_: arquivo, blob, attachment, Artefato

**Canal**:
Capacidade de enviar e receber no WhatsApp (whatsmeow em produção; stub nos testes e com `WHATSAPP_STUB=1`).
_Avoid_: bot, Cloud API, webhook (como sinónimo)

**Allowlist**:
Lista de JIDs de Conversa que recebem Ack e `POST /envios`. Fora dela o gateway **persiste** e **não** envia Ack nem aceita `POST /envios`. A **Resposta automática** pode enviar mesmo assim. O backoffice lista todas; só mostra compositor quando `permitido`.
_Avoid_: whitelist, ACL genérica

**Ack**:
Texto estático (`WHATSAPP_ACK_TEXTO`) enviado na Conversa da Allowlist após uma Mensagem viva (não FromMe, não Status, não histórico). Prova de canal; não consulta Oferta.
_Avoid_: bot reply, confirmação de leitura

**Resposta automática**:
Envio por substring no nome da Conversa (`WHATSAPP_AUTO_NOME` / `WHATSAPP_AUTO_TEXTO`; needle vazia = desligada). Na conversa **direta**: casa o `PushName` da Mensagem; no **grupo**: casa o assunto (`ConversaNome` no adapter). Case-insensitive, sem folding de acento. Recorte vivo do Ack, sem reacção / revogação / indecifrável. Substitui o Ack na mesma entrada. Não exige Allowlist.
_Avoid_: bot, ausência, vacation
