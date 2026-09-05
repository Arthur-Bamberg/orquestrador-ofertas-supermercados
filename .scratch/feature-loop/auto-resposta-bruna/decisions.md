# Decisões — auto-resposta-bruna

## Contexto

O gateway já persiste toda Mensagem e envia um **ack estático** (`WHATSAPP_ACK_TEXTO`) só para Conversas na allowlist (`deveAck`: não FromMe, não Status, não histórico). Conversa **não** tem campo Nome; o backoffice rotula com JID + `PushName` da última Mensagem. Pedido: responder automaticamente **"Estou trabalhando, não posso no momento"** em **todas** as Conversas cujo nome contenha **Bruna**.

## Lista

### D1 — O que conta como “nome da Conversa”
- **Pergunta:** Onde procurar “Bruna”?
- **Opções:**
  - A: Só `PushName` da Mensagem que acabou de chegar (é o que o backoffice já trata como rótulo).
  - B: Nome persistido em `Conversa` (coluna nova; assunto do grupo + nome do peer na direta).
  - C: **Título da Conversa no evento** + **PushName na direta**. Direta: casa `PushName`. Grupo: casa o **assunto do grupo** (whatsmeow `GroupInfo` / history `Conversation.Name` no adapter → `Entrada.ConversaNome`). Não varrer PushNames antigos da thread (uma mensagem da Bruna não “contagia” o grupo inteiro). Não casar o corpo da Mensagem.
- **Recomendado:** C — “nome **na conversa**” é o título do chat, não o texto nem “alguém chamado Bruna já falou aqui”. A é cega a grupo titulado “Bruna …” quando outra pessoa fala; B é migration + UI sem necessidade para disparar o envio (o adapter já vê o assunto).
- **Status:** confirmado

### D2 — Allowlist vs “todas”
- **Pergunta:** A resposta por nome respeita a allowlist (ADR do app 0001: allowlist = quem recebe ack / `POST /envios`)?
- **Opções:**
  - A: Só envia se a Conversa também estiver na allowlist (só muda o texto).
  - B: Envia para **qualquer** Conversa que D1 case, **fora** da allowlist. `POST /envios` continua allowlist. Ack de ofertas continua allowlist.
  - C: Acrescentar JIDs da Bruna à `WHATSAPP_ALLOWLIST` e reusar o ack (texto único para todos os permitidos).
- **Recomendado:** B — o pedido é “todas as conversas” com esse nome, não “as que já estão na lista”. C misturaria o ack de ofertas com ausência e exigiria JID manual. Ban: superfície extra de envio, mas o marcador é um nome próprio, não o mundo.
- **Depende de:** D1, D5.
- **Status:** confirmado
- **Nota:** se B confirmar, ADR curto no app na implementação (excepção explícita ao 0001).

### D3 — Marcador e texto: onde configuram
- **Pergunta:** “Bruna” e a frase ficam no código ou no env (raiz, como o resto do gateway)?
- **Opções:**
  - A: Hardcode no Go (`Bruna` + a frase).
  - B: `WHATSAPP_AUTO_NOME` (substring) e `WHATSAPP_AUTO_TEXTO`. Needle **vazia no código** = feature desligada (testes/clone não disparam). `.env.example` documenta `Bruna` e `Estou trabalhando, não posso no momento`; o `.env` local ganha os mesmos valores nesta entrega.
  - C: Lista CSV de vários nomes / várias regras.
- **Recomendado:** B — o pedido é pessoal e específico, mas nome em binário impede desligar e mudar sem rebuild. C é YAGNI (um marcador).
- **Status:** confirmado

### D4 — Quando dispara (frequência e tipos)
- **Pergunta:** Cada Mensagem viva que casa, ou com folga?
- **Opções:**
  - A: Cada entrada viva que D1 case — mesmo recorte do ack: não FromMe, não Status, não `origem=historico`, não duplicada (`provedor_id`). Não reagir a reacção / revogação / indecifrável.
  - B: No máximo uma vez por Conversa (flag no schema).
  - C: Cooldown (ex. 1 h) por Conversa.
- **Recomendado:** A — zero estado novo; simétrico ao ack. B/C são produto de “ausência” mais polido; o pedido não pediu. Risco: duas mensagens seguidas = duas respostas (igual ao ack actual).
- **Depende de:** D1.
- **Status:** confirmado

### D5 — Relação com o ack de ofertas
- **Pergunta:** Se a Conversa casa Bruna **e** está na allowlist, quantas saídas?
- **Opções:**
  - A: Uma só: **Resposta automática** substitui o ack nessa Mensagem.
  - B: As duas (ack + ausência).
  - C: Desligar o ack global; só resposta por nome.
- **Recomendado:** A — uma saída por entrada. Allowlisted sem o nome continua com `WHATSAPP_ACK_TEXTO`. C mataria a prova de canal nas outras Conversas.
- **Depende de:** D2, D3.
- **Status:** confirmado

### D6 — Onde vive a regra
- **Pergunta:** Camada e vocabulário?
- **Opções:**
  - A: Só no adapter whatsmeow (if PushName…).
  - B: Caso de uso `Receber` no `application` (ao lado de `deveAck`): porta `Canal` + persistência da saída iguais ao ack. Adapter só preenche `Entrada.ConversaNome` quando souber o assunto do grupo.
  - C: Worker/`agente-ofertas-worker` (ainda não existe).
- **Recomendado:** B — comportamento observável nos testes de `Receber` já existentes; C não existe; A fura o domínio e dificulta stub.
- **Glossário:** **Resposta automática** (por nome) ≠ **Ack** (allowlist). Evitar “bot”, “ausência” como entidade, “vacation”.
- **Status:** confirmado

### D7 — Directa vs grupo vs Status
- **Pergunta:** Em grupo, para quem envia? Status?
- **Opções:**
  - A: Mesmo destino do ack: JID da **Conversa** (grupo recebe no grupo). Status nunca (já fora do ack).
  - B: Se o grupo casa, responder em 1:1 ao remetente (não no grupo).
  - C: Só Conversas **diretas**; grupos ignorados mesmo com assunto “Bruna”.
- **Recomendado:** A — “conversas” no pedido inclui grupo cujo **título** tem Bruna. B seria surpresa no telemóvel do remetente. C estreita demais se existir grupo com esse nome.
- **Depende de:** D1.
- **Status:** confirmado

### D8 — Matching da substring
- **Pergunta:** Como casar “Bruna”?
- **Opções:**
  - A: Igualdade exacta, case-sensitive.
  - B: Contém, **case-insensitive** (`strings.ToLower`), sem folding de acento (`Bruná` não casa). “Bruna Silva” casa; “Bruno” não.
  - C: Regex / word-boundary.
- **Recomendado:** B — perfil WhatsApp e assunto de grupo raramente são o token isolado. C é overkill.
- **Depende de:** D3.
- **Status:** confirmado

### D9 — Backoffice, HTTP e persistência extra
- **Pergunta:** UI / coluna `conversa.nome` / rotas novas?
- **Opções:**
  - A: Nada neste slice: dispara no gateway; GET/backoffice inalterados.
  - B: Persistir Nome e mostrar no SPA (`conversaRotulo`).
  - C: Toggle no backoffice para ligar/desligar a regra.
- **Recomendado:** A — o pedido é responder no WhatsApp, não rotular a lista. B/C são features seguintes (B ainda ajuda o rastro).
- **Status:** confirmado

### D10 — Documentação
- **Pergunta:** O que actualizar além do código?
- **Opções:**
  - A: `CONTEXT.md` (Resposta automática vs Allowlist/Ack), `AGENTS.md` do app, `.env.example` raiz. ADR de app só se D2=B.
  - B: Só `.env.example`.
  - C: CONTEXT-MAP / ADR de workspace.
- **Recomendado:** A — vocabulário do canal muda (segundo motivo de envio). Workspace ADR não: não muda fronteira de apps.
- **Depende de:** D2, D6.
- **Status:** confirmado

## Revisão global
- Data/hora da passagem 2: 2026-08-31 ~16:05 America/Sao_Paulo
- O que mudou vs passagem 1:
  - D1 fechou em **não** persistir Nome e **não** usar PushName de mensagens antigas no grupo (evita spam em “família” onde a Bruna já falou).
  - D2 deixou `POST /envios` na allowlist; só a Resposta automática fura a lista.
  - D3: needle vazia por omissão no binário (não default Go `Bruna`), valores do pedido no `.env` / `.env.example`.
  - D4: reacção/revogação/indecifrável fora (o ack actual ainda pode disparar nesses tipos — não alargar o buraco).
  - D9 recusou schema/UI neste slice.
  - Uniu “destino do envio” em D7; matching de string em D8.
- Riscos remanescentes:
  - Assunto de grupo pode vir vazio se o store whatsmeow ainda não tiver `GroupInfo` — grupo titulado Bruna não dispara até haver nome.
  - Substring: um grupo “Bruna e amigos” dispara a **cada** mensagem de qualquer membro (D4=A).
  - D2=B aumenta envios fora da allowlist (ToS / ban, número dedicado já assumido no ADR 0007).

## Confirmação

- 2026-08-31: lote aceito sem overrides (D1–D10).
