# Decisões — gateway-whatsapp

## Contexto

Criar o app `apps/gateway-whatsapp` neste monorepo: processo Go longo que fala WhatsApp, persiste o rastro do canal e deixa um contrato para o `agente-ofertas-worker` (ainda não implementado). O catálogo (Oferta, Mercado, …) já existe em Postgres via `modules/ofertas-store`; o workspace ADR 0006 manda estado de canal **fora** das tabelas de Oferta/Documento. O roadmap Notion (“Ajudante do Mercado”, Nível 2) previa bot + matcher no mesmo processo com `whatsmeow`; o `AGENTS.md` da raiz **já desacoplou** gateway, agente e MCP — esta feature segue o workspace, não o Nível 2 gordo do Notion.

## Lista

### D1 — Onde vive o deployable
- **Pergunta:** O gateway é um módulo novo ou entra noutro app?
- **Opções:**
  - A: `apps/gateway-whatsapp` (Go 1.24, `go.work`, `cmd/gateway-whatsapp`, Clean Architecture como os outros apps)
  - B: Endpoints/worker dentro de `ofertas-api`
  - C: Serviço Node/Python ao lado do workspace
- **Recomendado:** A — já está na tabela de apps planejados; ADR 0001 / 0005 recusam misturar deployables (cron CLI vs HTTP vs canal 24/7).
- **Status:** confirmado

### D2 — Fronteira com agente, MCP e catálogo
- **Pergunta:** O gateway interpreta lista de compras / consulta Oferta, ou só transporta o canal?
- **Opções:**
  - A: Gateway **fino**: receber/enviar texto e mídia, Contato, Conversa, Mensagem, allowlist. **Não** importa `ofertas-store`. Não chama Extrator nem interpreta lista/imagem. Agente e MCP ficam para features seguintes.
  - B: Nível 2 do Notion: matcher/IA **in-process** (canais Go) neste mesmo binário
  - C: Gateway consulta `ofertas-api` / SQL do catálogo e formata a resposta (agente “escondido”)
- **Recomendado:** A — o `AGENTS.md` já nomeou três apps; B/C atrasam o desacoplamento e vazam Oferta para o canal. Entregável desta feature: canal demonstrável (ack), não comparativo de preços.
- **Depende de:** D1.
- **Status:** confirmado

### D3 — Transporte WhatsApp
- **Pergunta:** Qual protocolo/cliente?
- **Opções:**
  - A: [`whatsmeow`](https://github.com/tulir/whatsmeow) (WhatsApp multi-device, QR no terminal) — já escrito no roadmap Notion
  - B: WhatsApp Cloud API (Meta): webhooks, templates, janela de 24h, conta Business
  - C: Sidecar (Evolution API / similar) e o app só fala HTTP
- **Recomendado:** A — alinhado ao Notion, nativo em Go, sem WABA/templates neste estágio. Número **dedicado** (não o pessoal principal): risco de banimento da conta. B é o caminho de produto público depois; C acrescenta Redis/Compose sem ganho agora.
- **Consequência:** processo **sempre ligado** (TCP persistente). Cloud Run scale-to-zero da página Notion **não** serve para este transporte.
- **Status:** confirmado

### D4 — Quem pode falar com o canal
- **Pergunta:** Qualquer JID ou só uma allowlist?
- **Opções:**
  - A: Allowlist por env (`WHATSAPP_ALLOWLIST`, E.164 / JID), poucos testers. Fora da lista: **silêncio** (não persiste, não responde) — opt-in de facto e menos anúncio do bot
  - B: Aberto a quem mandar mensagem
  - C: Um único número hardcoded no código
- **Recomendado:** A — MVP pessoal/testers; B é risco de ban + LGPD; C não escala nem a um segundo celular de teste. Com D10, a lista inclui JIDs de **Conversa**: pessoa (`…@s.whatsapp.net`) e grupo (`…@g.us`). Mensagem de grupo só entra se o JID do **grupo** estiver na lista — remetente allowlisted noutro grupo não abre o canal.
- **Depende de:** D3 (cliente não oficial pede superfície mínima).
- **Status:** confirmado

### D5 — Persistência de Contato e Mensagem
- **Pergunta:** Onde mora o estado operacional do canal (não o catálogo)?
- **Opções:**
  - A: Mesmo Postgres (`DATABASE_URL`), **schema `whatsapp`**, migrations **do app** com tabela `whatsapp.schema_migrations` (não reutilizar `schema_migrations` do `ofertas-store`)
  - B: Tabelas novas **dentro** de `modules/ofertas-store`
  - C: Só sqlite (mensagens + sessão)
- **Recomendado:** A — ADR 0006 (mesmo banco, tabelas de canal separadas); o store do catálogo não ganha segundo consumidor de WhatsApp ainda (regra lazy de `modules/`). C mistura sessão da lib com histórico nosso e dificulta o agente ler Mensagem depois.
- **IDs:** opacos (UUID) na criação, como ADR 0018 do scraper; chave natural do Contato e da Conversa = JID.
- **Envio:** não há tabela `envio` à parte — status na Mensagem de saída (`pendente` / `enviado` / `falhou`). Idempotência de entrada: ID da mensagem no WhatsApp.
- **Mídia:** bytes em disco (`.data/midia/`), path na Mensagem — não bytea.
- **Status:** confirmado

### D6 — Sessão do dispositivo (pairing whatsmeow)
- **Pergunta:** Onde o whatsmeow guarda o device store (QR uma vez só)?
- **Opções:**
  - A: Postgres (`sqlstore` da lib no schema `whatsapp`)
  - B: sqlite em `apps/gateway-whatsapp/.data/` (gitignored), caminho no `.env`
  - C: Sem persistência de sessão (QR a cada start)
- **Recomendado:** B — caminho usual da lib, independente das migrations nossas; restart no VPS não pede QR se o ficheiro permanecer. A acopla upgrades do whatsmeow ao schema do app cedo demais. C é inviável em deploy.
- **Passagem 1 previa A; passagem 2 mudou para B** (ver Revisão global).
- **Status:** confirmado

### D7 — Contrato com o agente (ainda inexistente) e prova do canal
- **Pergunta:** Como entra mensagem e como sai resposta, sem construir o worker agora?
- **Opções:**
  - A: Caso de uso `ReceberMensagem` / `EnviarMensagem`. Porta `Canal` (whatsmeow em prod; stub nos testes). Sem agente configurado: **ack estático** (texto no `.env`, pt-BR) na mesma use case de envio. HTTP `POST /envios` (token) para o agente futuro usar o mesmo caso de uso. Sem NATS/Redis pub-sub nesta feature.
  - B: Canais Go in-memory + matcher no processo (Notion Nível 2)
  - C: Só persistir entrada; não responder nada até existir o agente
- **Recomendado:** A — demonstra o canal de ponta a ponta; C não prova envio; B viola D2. NATS fica para quando gateway e worker forem dois processos de verdade e o poll/HTTP apertar (Notion Nível 3). Ack vai para a mesma Conversa (direta ou grupo). `POST /envios` aceita texto e mídia.
- **Depende de:** D2, D5, D9.
- **Status:** confirmado

### D8 — Glossário do contexto Canal (não misturar com o catálogo)
- **Pergunta:** Quais termos são canônicos neste app?
- **Opções:**
  - A: **Contato** (pessoa no canal, JID). **Conversa** (direta 1:1 ou grupo; JID do chat). **Mensagem** (entrada/saída; corpo texto e/ou **Mídia**). Sem Oferta/Lista/Matcher aqui. Evitar: bot, chat, user, session (sessão do **dispositivo** é infra, D6).
  - B: Reusar “Usuário” / “Chat” / “Bot” do Notion
  - C: Sem Conversa (só Contato) — inválido com grupos (D10)
- **Recomendado:** A, **ajustado pelo override D10** — Conversa distingue direta de grupo; Mídia é parte da Mensagem, não entidade de catálogo. Lista de compras continua no agente.
- **Status:** confirmado (ajuste D10)

### D9 — Superfície HTTP e autenticação
- **Pergunta:** O processo expõe HTTP? Com auth? (`ofertas-api` é local e sem auth.)
- **Opções:**
  - A: Sim: `GET /health`, `GET /ready` (whatsmeow conectado), `POST /envios` com `Authorization: Bearer` / `GATEWAY_TOKEN`. Sem auth = 401. Não expor CRUD de Mensagem no MVP.
  - B: Zero HTTP (só WhatsApp + logs)
  - C: Mesma política da API: HTTP aberto sem token
- **Recomendado:** A — health para hospedar; send é a costura com o agente; C é perigoso num VPS. B atrasa o worker sem necessidade (o POST é fino).
- **Código vs plano:** desvio **explícito** do “no auth” da ofertas-api (ADR 0005 era MVP local).
- **Status:** confirmado

### D10 — Conteúdo e grupos
- **Pergunta:** O que o MVP aceita no WhatsApp?
- **Opções:**
  - A: Só **texto** em conversa **1:1**. Grupo: ignorar. Mídia/áudio/stickers: ignorar, sem resposta.
  - B: Texto + imagem (encarte na conversa)
  - C: Participar de grupos
  - D: Texto **e** mídia (imagem, áudio, vídeo, documento, figurinha) em conversa **direta e grupo**
- **Recomendado:** A — lista de compras no Notion é texto; mídia e grupos puxam o agente e o risco de spam.
- **Status:** override — D (pedido explícito). Gateway persiste e reenvia; **não** OCR/interpreta mídia. Allowlist pelo JID da Conversa (D4).


### D11 — Testes e WhatsApp ao vivo
- **Pergunta:** Como crescer com TDD sem Meta/conta real nos testes do hook?
- **Opções:**
  - A: Unidade: porta `Canal` in-memory + Postgres efémero **do próprio app** (não `storetest` do catálogo, que abre Oferta e dá truncate no catálogo). Live opt-in `LIVE_WHATSAPP=1` (fora do pre-commit), análogo a `LIVE_EXTRATOR`.
  - B: Testes só contra whatsmeow real
  - C: Reusar `modules/ofertas-store/storetest` e pendurar tabelas `whatsapp` no mesmo Open
- **Recomendado:** A — workspace TDD; B não roda no hook; C acopla o canal ao migrate/truncate do catálogo sem o gateway ser consumidor do store.
- **Status:** confirmado

### D12 — Documentação no repo
- **Pergunta:** O que atualizar além do código?
- **Opções:**
  - A: `apps/gateway-whatsapp/AGENTS.md` + `CONTEXT.md` (glossário D8). `CONTEXT-MAP.md` na raiz (Catálogo → scraper CONTEXT; Canal WhatsApp → este CONTEXT). `AGENTS.md` raiz: app passa a **Current**. ADR de workspace curto (whatsmeow + schema `whatsapp` + não Cloud Run). Sem ecrãs no backoffice.
  - B: Só um README no app
  - C: CONTEXT único na raiz misturando Oferta e Contato
- **Recomendado:** A — formato grill-with-docs; dois contextos de verdade. ADR só na implementação, se confirmarmos D3/D5 (hard-to-reverse + surpreende quem espera Cloud API).
- **Status:** confirmado

### D13 — Forma de correr e empacotar
- **Pergunta:** Como se opera local e na nuvem?
- **Opções:**
  - A: Binário longo (`go run ./cmd/gateway-whatsapp`). QR no stdout na primeira sessão. `Dockerfile` no app (ADR 0004: imagem quando o app é hospedado — este **é** 24/7). Compose da raiz **não** acrescenta o serviço neste slice (QR e `.data` no host são mais simples). TZ `America/Sao_Paulo`.
  - B: Serviço no `docker-compose.yml` já nesta feature
  - C: Cloud Run / scale-to-zero
- **Recomendado:** A — C contradiz D3; B pode vir depois se o sqlite da sessão for volume explícito.
- **Status:** confirmado

## Revisão global

- Data/hora da passagem 2: 2026-08-29 ~11:05 America/Sao_Paulo
- O que mudou vs passagem 1:
  - D6: sessão whatsmeow **sqlite em `.data/`**, não Postgres (a lib e o pairing são o caminho batido; Mensagem continua no PG).
  - D2/D7: recusa explícita do Nível 2 Notion (matcher no gateway); ack estático como prova de canal.
  - D5: Envio não é tabela — status na Mensagem; migrations isoladas de `schema_migrations` do catálogo.
  - D9: desvio consciente do no-auth da ofertas-api; `/health` + `/ready`.
  - D11: **não** reusar `storetest` do catálogo.
  - D13: Cloud Run scale-to-zero da página Notion marcado como incompatível com whatsmeow.
  - D8: sem entidade Conversa; lista de compras não é deste contexto.
- Riscos remanescentes:
  - ToS / banimento (whatsmeow, número dedicado).
  - Agente ainda não existe: o ack não substitui comparativo de preços (próximo `/feature-loop`).
  - Backup do sqlite de sessão é tão crítico quanto o Postgres para não re-parear.
  - `POST /envios` precisa de rede interna ou token forte se o processo for público.

## Confirmação

- 2026-08-29: lote aceito; **D10 override D** (grupo + mídia). D8 passa a incluir Conversa e Mídia.
