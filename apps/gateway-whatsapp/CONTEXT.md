# Gateway WhatsApp

Canal WhatsApp do workspace de ofertas: recebe e envia Mensagens (texto e Mídia) em Conversas diretas, sem consultar Oferta nem interpretar lista de compras.

## Language

**Contato**:
Pessoa no canal, identificada pelo JID canónico de telefone (`…@s.whatsapp.net`) a partir do número que a Cloud API entrega (`wa_id`). Boas-vindas e Aceite são deste Contato, não da Conversa.
_Avoid_: usuário, cliente, user, sender (como entidade)

**Boas-vindas**:
Fato no Contato de que o Canal já enviou a apresentação (Pague Menos Mercado, Termos de Uso e o pedido de Aceite com 1 ou 2). Uma vez por Contato, na Conversa da primeira Mensagem viva de texto; nessa Mensagem o Canal não lê `1` — ainda não houve apresentação.
_Avoid_: bem-vindo, welcome, flag de bem-vindo, apresentação (como entidade persistida)

**Termos de Uso**:
Texto jurídico apresentado nas Boas-vindas, com o qual o Contato faz o Aceite. Não é o Termo do Agente (busca do Item). Não volta no Pedido de Aceite.
_Avoid_: Termo (sem “de Uso”), ToS, política de privacidade (como se fosse este texto)

**Pedido de Aceite**:
Mensagem curta depois das Boas-vindas, ainda sem Aceite: o uso exige Aceite e pergunta 1 ou 2. Não republica os Termos de Uso. Não é Consulta nem Ack.
_Avoid_: nag, lembrete, boas-vindas (como se fosse de novo)

**Aceite**:
Concordância do Contato com os Termos de Uso; uma vez no Contato, vale em toda Conversa desse Contato. Só o texto `1` concede; `2` e qualquer outro corpo reiteram com Pedido de Aceite. Sem Aceite o texto não é Lista, Consulta nem Recusa.
_Avoid_: termo-aceito, consentimento, opt-in, usuário aceitou, recusa dos Termos (como fato persistido)

**Conversa**:
Thread **direta** (1:1) entre o Canal e um Contato. O rastro antigo pode ainda ter linhas de grupo ou Status; o Canal deste recorte não as ingere.
_Avoid_: chat, sala, thread, grupo, Status (como recorte actual)

**Mensagem**:
Unidade persistida de entrada ou saída numa Conversa: corpo de texto, Mídia opcional, id do provedor (idempotência), origem (`vivo` ou, no rastro antigo, `historico`), timestamp do WhatsApp e, na saída, status de envio (`pendente`, `enviado`, `falhou`, `entregue`, `lido`, `reproduzido`). Inclui reacção. Status de envio **pendente** não é o estado do Canal.
_Avoid_: evento, payload, Envio (como tabela), Canal pendente (como se fosse este status)

**Mídia**:
Anexo da Mensagem — imagem, áudio, vídeo, documento ou figurinha — bytes no disco do gateway quando o download consegue, metadados na Mensagem mesmo se falhar. O gateway não interpreta o conteúdo.
_Avoid_: arquivo, blob, attachment, Artefato

**Canal**:
Capacidade de enviar e receber no WhatsApp (Cloud API em produção; stub nos testes e com `WHATSAPP_STUB=1`). Um por processo; não pertence a um Mercado. O nome comercial do produto (hoje Pague Menos Mercado) não é um Canal. Estado: **pronto** (credenciais da conta Business presentes, ou stub local) ou **não configurado** (falta token, número ou verify). Quando pronto, o Canal tem o JID do Número do Canal — visível ao Operador; não é um Contato. **Pronto** aqui não é o status de envio da Mensagem.
_Avoid_: bot, webhook (como sinónimo), Mercado, Pague Menos Mercado (como se fosse este Canal), pareamento, QR, whatsmeow, websocket

**Número do Canal**:
Número WhatsApp Business verificado da conta, identidade visível do Canal ao Operador (JID `…@s.whatsapp.net`). Não é um Contato.
_Avoid_: telefone dedicado (como se fosse aparelho pareado), JID de Contato, phone number id (como termo de domínio)

**Janela**:
Intervalo de 24 horas após a última Mensagem viva de entrada da Conversa, durante o qual o Canal envia texto e Mídia livres. Sem Janela aberta, a saída livre não parte; entra Template.
_Avoid_: sessão (como entidade), 24h (sem nome), conversation window (como termo canónico)

**Template**:
Mensagem aprovada na conta Business, usada para sair quando a Janela está fechada. Não é os Termos de Uso nem o Termo do Agente.
_Avoid_: HSM, mensagem livre, ack, modelo (sem este sentido)

**Allowlist**:
Lista de JIDs de Conversa em que o Operador envia texto pelo backoffice (`permitido` no resumo; `POST /envios` com cookie de Operador). Não porta Boas-vindas, Pedido de Aceite, Ack, chamada ao Agente nem `POST /envios` com Bearer do Agente.
_Avoid_: whitelist, ACL genérica

**Ack**:
Texto estático (`WHATSAPP_ACK_TEXTO`) após Mensagem viva (não histórico) **quando o Agente não está ligado** e o Contato remetente tem Aceite. Prova de canal; não consulta Oferta. Sem Aceite saem Boas-vindas ou Pedido de Aceite, não Ack. Com Agente, a Resposta substitui o Ack.
_Avoid_: bot reply, confirmação de leitura
