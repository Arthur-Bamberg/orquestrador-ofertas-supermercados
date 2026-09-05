# Decisões — persistir-mensagens-whatsapp

## Contexto

O gateway **já** tem Contato / Conversa / Mensagem, schema `whatsapp` e `SalvarMensagem`. Em 2026-08-30 as três tabelas estavam **vazias**: allowlist impedia INSERT, `FromMe`/vazio caíam no adapter, e `events.HistorySync` (já disparado no pairing) **não tinha handler**. O whatsmeow entrega o arquivo em `HistorySync.Conversations` + `statusV3Messages`; o cliente ignora esse evento.

**Diretriz do usuário (2026-08-30):** ter o **máximo de dados possíveis** do WhatsApp no Postgres — não um rastro mínimo para o agente.

Glossário: `apps/gateway-whatsapp/CONTEXT.md`. D4 antigo (silêncio = não persiste) fica **override**.

## Lista

### D1 — Allowlist vs persistência
- **Pergunta:** A allowlist ainda impede o INSERT?
- **Opções:**
  - A: Só persiste Conversa na lista (D4 antigo).
  - B: Persiste toda **direta** ao vivo; grupo só na lista.
  - C: Persiste **toda** Conversa (direta, grupo, Status) de conteúdo de Mensagem (D11). Allowlist **só** para ack estático e `POST /envios`.
- **Recomendado:** C — pedido explícito de máximo rastro; silêncio no WhatsApp (não ack o mundo) continua a proteger ban.
- **Status:** override — C (usuário: máximo de dados); confirmado 2026-08-30
- **Nota:** actualizar Allowlist no `CONTEXT.md` após confirmação.

### D2 — History sync
- **Pergunta:** O blob `events.HistorySync` entra no Postgres?
- **Opções:**
  - A: Não (só `events.Message` ao vivo).
  - B: Ingerir history só das Conversas que D1 deixaria passar.
  - C: Ingerir **todo** o blob: `Conversations` (via `ParseWebMessage`, o próprio whatsmeow documenta este caminho) **e** `statusV3Messages`. Idempotência pelo `provedor_id`.
- **Recomendado:** C — é onde está o arquivo que o pairing já descarregou; sem handler, esses chunks são lixo. Os chunks **já despachados nesta sessão** não voltam (whatsmeow apaga o media no servidor depois do dispatch). Depois do deploy: ou novos chunks, ou `BuildHistorySyncRequest` on-demand, ou re-parear (D12).
- **Status:** override — C (máximo de dados); confirmado 2026-08-30
- **Depende de:** D1, D12.

### D3 — JID gravado (PN e LID)
- **Pergunta:** Qual JID é chave de Contato/Conversa?
- **Opções:**
  - A: String crua do evento.
  - B: Só PN canónico; LID só se não houver telefone.
  - C: Colunas `jid` (PN canónico quando existir) **e** `jid_lid` (LID sem device). Unique em ambos (NULLs distintos no LID). Upsert encontra por PN **ou** LID para não duplicar Conversa.
- **Recomendado:** C — máximo de dados + evita duplicar Conversa quando o próximo evento traz o outro identificador. Passagem 1 pedia B (lazy); o override de volume torna o mapping no schema justificado.
- **Status:** confirmado

### D4 — Matching da allowlist (ack / envios)
- **Pergunta:** Como a lista no `.env` reconhece a Conversa se já não filtra INSERT?
- **Opções:**
  - A: Igualdade exacta actual (LID/device/9º dígito BR falham).
  - B: Strip de device; candidatos chat+PN+LID; variante BR do 9 após `55`+DDD; grupo `@g.us` sem inventar dígitos.
  - C: Operador cola `@lid` cru.
- **Recomendado:** B — sem isto o ack continua morto mesmo com Mensagem no banco. Lista = quem **recebe resposta**, não quem é gravado.
- **Status:** confirmado

### D5 — FromMe, vazio, tipos que hoje caem
- **Pergunta:** O que ainda se descarta?
- **Opções:**
  - A: Ignorar `FromMe`, vazio, não-`events.Message`, Status.
  - B: Persistir `FromMe` como Mensagem de **saída** (sem reenviar no Canal). Corpo vazio **com** Mídia, reacção, edit ou revogação ainda é Mensagem (D11). Protocolo sem conteúdo útil continua de fora.
  - C: Incluir reacções/protocolo **e** presença/digitando.
- **Recomendado:** B — máximo de conteúdo de Conversa; C mistura sinalização (D11). `Enviar`/`POST /envios` continua a gravar saída; `FromMe` ao vivo/histórico usa o mesmo `provedor_id` (idempotente, não duplica).
- **Status:** override — B (FromMe + não-texto; não presença); confirmado 2026-08-30

### D6 — Logs
- **Pergunta:** Observabilidade?
- **Opções:**
  - A: Sem log.
  - B: Uma linha por persistência (id, jid, direcao, origem `vivo`/`historico`) e por drop residual (`protocolo`, `midia_falhou` parcial). Sem corpo. Chunk de history: contagem ingerida / saltada.
  - C: Prometheus.
- **Recomendado:** B
- **Status:** confirmado

### D7 — Mídia
- **Pergunta:** Download falhou ou mídia só no histórico?
- **Opções:**
  - A: Sem bytes e sem texto ⇒ descarta a Mensagem.
  - B: INSERT sempre; path vazio + log se o download falhar; texto/caption preservados.
  - C: Fila de retry.
- **Recomendado:** B — não perder a linha. C é slice à parte. Histórico: **tentar** download (D16); falha não aborta o INSERT.
- **Status:** confirmado (ajustado: nunca dropar a Mensagem por mídia)

### D8 — HTTP / backoffice
- **Pergunta:** UI nesta feature?
- **Opções:**
  - A: Não. SQL no schema `whatsapp`.
  - B: `GET /mensagens`.
  - C: Tela no backoffice.
- **Recomendado:** A — o pedido é dados no Postgres, não ecrã.
- **Status:** confirmado

### D9 — Testes
- **Pergunta:** Como travar sem Meta?
- **Opções:**
  - A: Unidade: normalização JID/LID/device/BR; `Receber` persiste direta **e** grupo fora da allowlist; `FromMe` → saída sem chamar Canal; idempotência `provedor_id`; extração `HistorySync` → lista de `Entrada` com fixture proto/JSON mínimo (não live); allowlist só impede ack. Store efémero se D3=C (colunas novas).
  - B: Live no hook.
  - C: Só `pg.Store`.
- **Recomendado:** A
- **Status:** confirmado (casos actualizados na passagem 3)

### D10 — Docs
- **Pergunta:** O que actualizar?
- **Opções:**
  - A: Só CONTEXT/AGENTS.
  - B: CONTEXT + AGENTS + ADR do app: rastro completo no schema `whatsapp`; allowlist só no envio; history ingerido. Workspace 0007 não muda de casa (Postgres canal), muda a **amplitude**.
  - C: Só README.
- **Recomendado:** B — hard-to-reverse, surpreende quem leu D4 antigo, trade-off LGPD vs arquivo.
- **Status:** confirmado

### D11 — Envelope: o que é “máximo” (e o que não é)
- **Pergunta:** Tudo o que o whatsmeow emite, ou só o que é rastro de Conversa/Mensagem?
- **Opções:**
  - A: **Mensagem-shaped:** texto, Mídia, caption, `FromMe`, grupos, Status (`status@broadcast` / `statusV3Messages`), reacção, edit (update pelo `provedor_id`), revogação, `UndecryptableMessage` (linha `indecifravel`, preenchida se o retry chegar), timestamp do **provedor** (`Info.Timestamp`), push name, origem `vivo`|`historico`. Extra em `jsonb` (`payload`: quoted id, emoji da reacção, flags view-once/ephemeral/edit). **Não** protobuf cru.
  - B: A + recibos (delivered/read/played) como update de status na Mensagem de saída.
  - C: B + presença, digitando, chamadas, pin/mute/archive.
- **Recomendado:** B — recibos são estado da Mensagem (já temos `status` de envio); C não é Mensagem e explode volume sem valor de arquivo. Protobuf cru acopla o schema à versão da lib.
- **Depende de:** D1, D5.
- **Status:** confirmado (novo na passagem 3)

### D12 — History já consumido neste pairing
- **Pergunta:** Os chunks desta sessão já foram dispatch + apagados no servidor. Como preencher o buraco?
- **Opções:**
  - A: Só eventos futuros; arquivo desta sessão perdido.
  - B: Depois de ligar, se o operador pedir (env `WHATSAPP_HISTORY_ON_DEMAND=1`) ou se `whatsapp.mensagem` estiver vazia, `BuildHistorySyncRequest` (chunks `ON_DEMAND`).
  - C: Exigir re-QR (apagar sqlite) como único caminho.
- **Recomendado:** B — não apagar o pairing por omissão; on-demand é o que a lib oferece. A deixa o Postgres vazio até a próxima mensagem ao vivo. C só como fallback documentado no AGENTS.
- **Depende de:** D2.
- **Status:** confirmado (novo na passagem 3)

### D13 — Tipo de Conversa Status
- **Pergunta:** Stories (`status@broadcast`) são Conversa no domínio?
- **Opções:**
  - A: Não gravar Status.
  - B: Conversa tipo `status` (terceiro valor além de `direta`/`grupo`), JID `status@broadcast`. Mensagens entram no mesmo `whatsapp.mensagem`.
  - C: Tabela à parte.
- **Recomendado:** B — máximo no mesmo rastro; C é schema extra sem consumidor. CONTEXT ganha o tipo.
- **Status:** confirmado (novo na passagem 3; D1=C implica isto)

### D16 — Bytes de Mídia no histórico
- **Pergunta:** Descarregar anexos de history (potencialmente GB) ou só metadados?
- **Opções:**
  - A: Só metadados no histórico; bytes só em Mensagem ao vivo.
  - B: Tentar download de **toda** Mídia (vivo e histórico), best-effort (D7).
  - C: Limite (N MB / N ficheiros) no histórico.
- **Recomendado:** B — alinhado a máximo de dados; disco em `MIDIA_ROOT`. Se o volume dores, C num slice seguinte (não cap agora).
- **Status:** confirmado (novo na passagem 3)

## Revisão global

- Data/hora da passagem 2: 2026-08-30 ~13:35 America/Sao_Paulo
- Data/hora da passagem 3 (override usuário): 2026-08-30 ~13:30 America/Sao_Paulo
- O que mudou vs passagem 2:
  - Diretriz **máximo de dados** → D1=C, D2=C, D5=B (já não B mínimo de 1:1).
  - D3 passou de B (só PN) para **C** (PN+LID no schema) porque o volume e os dois identificadores são o arquivo.
  - D7: nunca dropar a linha por falha de Mídia.
  - Novos: D11 envelope (Mensagem + recibos; não presença), D12 buraco do history já dispatch, D13 tipo `status`, D16 bytes históricos.
  - Ack/envios continuam allowlist (D4=B) — persistir ≠ responder.
- Riscos remanescentes:
  - Número **pessoal** pareado + history + grupos + Status no mesmo Postgres do catálogo (LGPD). Número dedicado (ADR 0007) continua o mitigador; o código não anonimiza.
  - Disco de Mídia histórica pode crescer muito (D16=B).
  - Chunks desta sessão já foram embora; sem D12=B o banco só enche daqui para a frente.
  - `UndecryptableMessage` pode ficar para sempre se o remetente não reenviar.
  - Recibos (D11=B) actualizam Mensagem de saída; entrada não tem `lido` neste slice (não há coluna). Aceitável: status de envio já existe.
