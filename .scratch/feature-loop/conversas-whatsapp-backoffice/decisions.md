# Decisões — conversas-whatsapp-backoffice

## Contexto

O gateway já persiste **todas** as Conversas/Mensagens no schema `whatsapp` (ADR do app 0001; allowlist só no ack/`POST /envios`). O backoffice hoje só consome `ofertas-api` (`VITE_API_BASE`, catálogo). O gateway HTTP só tem `GET /health`, `GET /ready` e `POST /envios` (Bearer). A UI do catálogo declara filtros que a API **ignora**: `documentos.dia`, `ofertas.texto`, `produtos.nome`, `produtos.categoria`. Esta feature liga o rastro do canal ao backoffice e faz esses query params filtrarem de verdade.

## Lista

### D1 — Quem serve a leitura HTTP do canal
- **Pergunta:** De onde o backoffice lê Conversa/Mensagem?
- **Opções:**
  - A: `ofertas-api` consulta o schema `whatsapp` (ou um `modules/` novo).
  - B: `gateway-whatsapp` ganha GET de leitura; o SPA fala com duas bases (`VITE_API_BASE` + `VITE_GATEWAY_BASE`).
  - C: nginx/Vite faz BFF same-origin (`/gateway` → `:8090`) e o SPA não conhece a porta do gateway.
- **Recomendado:** B — ADR 0007 rejeitou dobrar o canal na `ofertas-api`; o dono do schema continua a ser o gateway. C muda o padrão já usado com a API de catálogo (browser → `:8080` com CORS).
- **Status:** confirmado

### D2 — Auth nos GETs do canal
- **Pergunta:** `GET /conversas` (e mídia) exigem `GATEWAY_TOKEN`?
- **Opções:**
  - A: Bearer igual a `POST /envios` (o SPA precisaria do token de envio, ou um BFF).
  - B: GETs sem auth, no mesmo MVP local da `ofertas-api` (ADR 0005). `POST /envios` continua Bearer.
  - C: Só same-origin via proxy (D1=C).
- **Recomendado:** B — não colocar `GATEWAY_TOKEN` no Vite. Porta `:8090` já está exposta no Compose local, como `:8080`.
- **Depende de:** D1.
- **Status:** confirmado

### D3 — CORS e env do SPA
- **Pergunta:** Como o browser na `:5173` chama o gateway na `:8090`?
- **Opções:**
  - A: `VITE_GATEWAY_BASE` (padrão `http://localhost:8090`) + `CORS_ORIGIN` no gateway (já existe no `.env` da raiz para a API).
  - B: Proxy Vite + nginx `/gateway` sem CORS.
  - C: SPA continua só em `VITE_API_BASE` e a API faz proxy.
- **Recomendado:** A — mesmo desenho de `VITE_API_BASE`. Dockerfile do backoffice passa a bake-ar `VITE_GATEWAY_BASE`. Gateway `config` passa a ler `CORS_ORIGIN`.
- **Depende de:** D1=B, D2=B.
- **Status:** confirmado

### D4 — Superfície HTTP de leitura
- **Pergunta:** Quais rotas o gateway expõe?
- **Opções:**
  - A: `GET /conversas` e `GET /conversas/{id}/mensagens`.
  - B: A + `GET /mensagens/{id}/midia` (bytes via `MidiaStore`, keyed por id da Mensagem — sem path cru na URL).
  - C: B + CRUD de Contato e envio pela UI.
- **Recomendado:** B — arquivo de Conversa sem Mídia fica cego. Envio reutiliza `POST /envios` (D14). Sem CRUD de Contato. Prefixos iguais aos actuais do gateway (`/health`, `/envios`): **sem** `/api`.
- **Status:** confirmado

### D5 — Contrato JSON
- **Pergunta:** Qual shape?
- **Opções:**
  - A: structs de `domain` (hoje `POST /envios` serializa PascalCase).
  - B: DTOs camelCase (`conversaId`, `criadoEm`, …), array cru como a `ofertas-api`.
  - C: envelope `{ items: [...] }`.
- **Recomendado:** B — o backoffice já fala camelCase com a API de catálogo. Não mudar o JSON de `POST /envios` neste slice.
- **Status:** confirmado

### D6 — Resumo na lista de Conversas
- **Pergunta:** O que cada linha da lista carrega?
- **Opções:**
  - A: só `id`, `jid`, `jidLid`, `tipo`.
  - B: A + última Mensagem (corpo, `criadoEm`, `pushName`, `direcao`, `tipo`) e `totalMensagens`.
  - C: embed de todas as Mensagens na lista.
- **Recomendado:** B — lista navegável sem N+1 no SPA. C explode com history sync. Novo método no `Repositorio` (`ListarConversas` / resumo); não poluir a entidade `Conversa`.
- **Status:** confirmado

### D7 — Volume da thread
- **Pergunta:** `GET /conversas/{id}/mensagens` devolve o arquivo inteiro?
- **Opções:**
  - A: todas, `ORDER BY criado_em` (comportamento actual de `ListarMensagens`).
  - B: as mais recentes, `limit` (default 200) e cursor `antes` (`criadoEm`+`id`); UI “carregar anteriores”.
  - C: `offset`/`page`.
- **Recomendado:** B — history de grupo pode ser enorme; “todas as Conversas” ≠ um JSON com 10k Mensagens. C é pior com inserts no meio. Lista de Conversas **não** pagina neste slice.
- **Depende de:** D4, D6.
- **Status:** confirmado

### D8 — UI no backoffice
- **Pergunta:** Como as Conversas entram no SPA?
- **Opções:**
  - A: `ResourceConfig` genérico (Novo / Editar / Apagar).
  - B: páginas **sem CRUD de rastro**: nav **Conversas**, lista + detalhe com Mensagens em ordem cronológica (`direcao`, `pushName`, corpo, link de Mídia). Compositor de texto no detalhe quando `permitido` (D14).
  - C: duas colunas estilo WhatsApp Web.
- **Recomendado:** B — A mentiria Novo/Editar/Apagar sobre um rastro que o gateway possui. C é produto extra; o resto do backoffice é lista+detalhe. Cliente HTTP **separado** do `request()` de catálogo (esse prefixa `/api`).
- **Depende de:** D1, D4, D5, D14.
- **Status:** confirmado (compositor no detalhe via D14)

### D9 — Filtros da lista de Conversas
- **Pergunta:** Como achar uma Conversa?
- **Opções:**
  - A: sem filtro (lista completa).
  - B: `tipo` (`direta` / `grupo` / `status`) e `q` (contém em `jid`, `jidLid` ou `pushName` da última).
  - C: B + “só allowlist”.
- **Recomendado:** B — o pedido é **todas** as Conversas; allowlist não entra no GET (exigiria vazar `WHATSAPP_ALLOWLIST` ou um flag persistido). Default: sem query = todas, ordenadas pela última Mensagem (mais recente primeiro).
- **Status:** confirmado

### D10 — Rótulo visível da Conversa
- **Pergunta:** O schema não tem subject de grupo. O que a lista mostra?
- **Opções:**
  - A: só JID.
  - B: JID sempre; subtítulo = `pushName` da última Mensagem quando houver (útil em direta; em grupo é o último autor, não o nome do grupo).
  - C: persistir subject/nome do grupo agora (query whatsmeow / evento extra).
- **Recomendado:** B — C é persistência nova, fora deste slice. Gap aceite: grupos aparecem como `…@g.us`.
- **Status:** confirmado

### D11 — Independência do canal ligado
- **Pergunta:** GET de rastro exige WhatsApp conectado (`/ready`)?
- **Opções:**
  - A: sim (503 se stub/desligado).
  - B: não — lê Postgres; `/ready` continua a ser só o canal.
- **Recomendado:** B — o arquivo já está no banco; o backoffice serve para inspecionar mesmo com stub ou QR pendente.
- **Status:** confirmado

### D12 — Quais filtros de catálogo corrigir
- **Pergunta:** A UI manda query params que a `ofertas-api` ignora. Quais passar a honrar?
- **Opções:**
  - A: só os quatro rotos: `GET /documentos?dia=`, `GET /ofertas?texto=`, `GET /produtos?nome=`, `GET /produtos?categoria=`.
  - B: A + expor na UI o `documentoId` de Oferta (já funciona na API, não está nos filtros do SPA).
  - C: filtrar no cliente e deixar a API como está.
  - D: remover da UI os filtros que a API não implementa.
- **Recomendado:** A — é o que o operador já vê “não funcionar”. B é extra. C/D empurram o contrato para o browser. Implementar **no handler da API** com o `filter` in-memory já usado (`mercadoId`, `fonteId`, …), sem mudar `modules/ofertas-store` neste slice. `texto` carrega `ListProdutos`/`ListMarcas` no handler para resolver nomes.
- **Status:** confirmado

### D13 — Semântica dos query params novos
- **Pergunta:** O que cada filtro significa?
- **Opções:** (único pacote recomendado)
  - `dia`: igualdade exacta com `Documento.dia` (`YYYY-MM-DD`, o que o `<input type="date">` envia).
  - `nome` (Produto): contém, sem distinção de maiúsculas, em `nome` **ou** `nomeNorm`.
  - `categoria`: contém, sem distinção de maiúsculas, nalgum elemento de `categorias` (o filtro da UI é texto livre, não select).
  - `texto` (Oferta): contém, sem distinção de maiúsculas, no `nome`/`nomeNorm` do Produto **ou** da Marca ligados. Não busca no UUID da Oferta nem no JSON de promoção.
- **Recomendado:** este pacote — alinhado aos campos que a UI já envia e ao glossário (Produto/Marca, não “search genérico”).
- **Depende de:** D12=A.
- **Status:** confirmado

### D14 — Mutações no canal a partir do backoffice
- **Pergunta:** O SPA envia Mensagem ou apaga Conversa?
- **Opções:**
  - A: não. Só leitura.
  - B: `POST /envios` a partir do detalhe (exigiria token no SPA ou BFF).
  - C: DELETE de Mensagem/Conversa no Postgres.
- **Recomendado:** A — o pedido original era ver as Conversas; envio continua allowlist + Bearer no gateway.
- **Status:** override — B (usuário 2026-08-31: implementar envio pelo backoffice). Não C (não apagar rastro).
- **Como (override):** o detalhe chama o `POST /envios` já existente (`conversaJid` + `corpo`, texto). O SPA manda `Authorization: Bearer` com `VITE_GATEWAY_TOKEN` (local; o catálogo já muta sem auth). CORS do gateway inclui `Authorization`. Allowlist **não** cai: o resumo da Conversa ganha `permitido` (boolean, calculado no processo com a allowlist em memória — não vaza a lista crua). Compositor só se `permitido`; 403 continua a ser o contrato se alguém forçar o POST. Sem envio de Mídia pela UI neste slice.

### D15 — Documentação
- **Pergunta:** O que actualizar depois de implementar?
- **Opções:**
  - A: só AGENTS do gateway e do backoffice.
  - B: A + `CONTEXT-MAP.md` (Canal → backoffice, leitura) + ADR curto de workspace: backoffice lê o canal **via HTTP do gateway**, não via `ofertas-api`.
  - C: só ADR.
- **Recomendado:** B — surpreende quem leu ADR 0005 (backoffice = catálogo) e 0007 (não dobrar o canal na API); o trade-off é real. `CONTEXT.md` do canal não muda de vocabulário; no máximo uma linha de que o rastro é legível por HTTP.
- **Status:** confirmado

## Revisão global

- Data/hora da passagem 2: 2026-08-31 ~07:40 America/Sao_Paulo
- O que mudou vs passagem 1:
  - D1: C (BFF nginx) deixou de competir de igual para igual com B; o Compose já ensina o browser a falar com `:8080` — repetir o padrão na `:8090` é menos surpresa.
  - D4 e mídia unificados numa superfície (não ficou “lista agora, bytes depois”). URL de mídia por **id da Mensagem**, não `midia_path` (path traversal).
  - D7 passou de “devolver tudo” para cursor das 200 mais recentes: “todas as Conversas” não implica payload único da thread.
  - D8 recusou Resource CRUD (passagem 1 flertava com `resources.ts`).
  - D12: `documentoId` em Ofertas saiu do escopo (API já filtra; a UI não promete).
  - D13 fechou a semântica de `texto` em Produto/Marca — Oferta não tem campo de texto próprio.
  - D11 novo na passagem 2: rastro ≠ canal ligado.
- Riscos remanescentes:
  - GETs sem auth na `:8090` expõem rastro e bytes de Mídia na rede local (mesmo nível LGPD do número pareado + history no Postgres).
  - Grupos sem subject: lista mostra JID `…@g.us`.
  - `ListOfertas` + `texto` continua a puxar todas as Ofertas e filtrar em memória (já é o padrão da API; volume local).
  - Thread com `limit=200` esconde o arquivo antigo até o operador carregar anteriores.
  - `POST /envios` continua PascalCase; só os GETs novos são camelCase — dois dialects no mesmo processo até um slice de contrato.
  - Override D14: `VITE_GATEWAY_TOKEN` fica no bundle do SPA (quem abre o backoffice pode enviar às Conversas da allowlist). Aceitável no MVP local; não é BFF.

## Confirmação

- Data: 2026-08-31 ~07:47 America/Sao_Paulo
- D1–D13, D15: confirmados como recomendado
- D14: override B (envio texto no detalhe; sem DELETE)
