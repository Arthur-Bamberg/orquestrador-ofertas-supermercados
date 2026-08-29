# Gateway WhatsApp

Canal WhatsApp do workspace de ofertas: recebe e envia Mensagens (texto e Mídia) em Conversas diretas ou de grupo, sem consultar Oferta nem interpretar lista de compras.

## Language

**Contato**:
Pessoa no canal, identificada pelo JID (telefone E.164 normalizado para `…@s.whatsapp.net`).
_Avoid_: usuário, cliente, user, sender (como entidade)

**Conversa**:
Thread do canal: **direta** (1:1, JID da pessoa) ou **grupo** (`…@g.us`). A allowlist vale sobre o JID da Conversa, não sobre o remetente isolado.
_Avoid_: chat, sala, thread

**Mensagem**:
Unidade persistida de entrada ou saída numa Conversa: corpo de texto, Mídia opcional, id do provedor (idempotência na entrada) e, na saída, status de envio (`pendente`, `enviado`, `falhou`).
_Avoid_: evento, payload, Envio (como tabela)

**Mídia**:
Anexo da Mensagem — imagem, áudio, vídeo, documento ou figurinha — bytes no disco do gateway, metadados na Mensagem. O gateway não interpreta o conteúdo.
_Avoid_: arquivo, blob, attachment, Artefato

**Canal**:
Capacidade de enviar e receber no WhatsApp (whatsmeow em produção; stub nos testes e com `WHATSAPP_STUB=1`).
_Avoid_: bot, Cloud API, webhook (como sinónimo)

**Allowlist**:
Lista de JIDs de Conversa autorizados. Fora dela o gateway permanece em silêncio (não persiste, não responde).
_Avoid_: whitelist, ACL genérica
