---
name: vitrine-coleta
description: Coleta de vitrine — adds a Mercado adapter to ofertas-scraper-v2 (TDD parser fixtures, Defaults registry, Mercado seed, Fonte tipo=site, live Coleta until every complete card is an Oferta). Use when adding a Mercado site Coleta, vitrine adapter, or Mercado seed in ofertas-scraper-v2.
---

# Coleta de vitrine

Um Mercado novo no `ofertas-scraper-v2` é um **adapter de Vitrine** + **seed**. A unidade de trabalho é a **Coleta** (termo do Item + Mercado + dia `America/Sao_Paulo`): busca HTTP no site até esgotar os cartões e persiste **Oferta de cada cartão completo**. Relacionados entram; Filtragem não é desta app. Fort é o molde. Encarte PDF fica no `ofertas-scraper`. Glossário: [`apps/ofertas-scraper/CONTEXT.md`](../../../apps/ofertas-scraper/CONTEXT.md). App: [`apps/ofertas-scraper-v2/AGENTS.md`](../../../apps/ofertas-scraper-v2/AGENTS.md). ADRs: 0012, 0013, 0014, 0015, 0040.

## Done

Coleta **`concluido`** do termo de prova neste Mercado: vitrine esgotada, uma Oferta por cartão completo, **zero** descarte. O site (browser ou o mesmo JSON/HTML da busca) mostra N cartões completos e o catálogo tem N Ofertas. Testes do módulo verdes. Mercado no seed do v2; Fonte `tipo=site` no seed do `ofertas-scraper` se ainda não existir.

`parcial` por corte (página seguinte falhou / tempo) **não** é Done — conserta paginação e retenta. Descarte de hit que no bruto tem preço e medida é parser. Hit cru sem preço/medida **depois** de mapear os campos reais: reporta e para (não chame de Done).

O usuário **nomeia** o Mercado. Sem nome, pergunta uma vez e para.

## Passos

Copie e marque:

```
- [ ] 1 Molde + plataforma
- [ ] 2 Fixture da busca
- [ ] 3 Parser TDD
- [ ] 4 Registro + seed
- [ ] 5 Coleta viva até Done
```

### 1. Molde + plataforma

Ler `internal/infra/vitrine/fort`, `osuper`, `vitrinehttp`, `registry.go`. Loja = **default do site** (Fort: não Canoas). Abrir o home do Mercado; achar a busca.

- HTML/JSON com `searchEngineToken` e `sense.osuper.com.br` → pacote irmão do Fort: `AccountID` + `LojaSenseID` do default, `osuper.SearchURL`, `osuper.ParseSearch`, `osuper.PageSize` (12), token no header como Fort. IDs ficam no pacote, não neste skill.
- Outro formato → `Parse` próprio → `domain.CartaoVitrine` + `vitrinehttp.Client`.

**Certo quando:** URL de busca, loja default e formato do cartão estão no pacote novo (ou reuso Osuper), com teste da URL/IDs.

### 2. Fixture da busca

Gravar **uma página real** da busca do termo de prova em `internal/infra/vitrine/<mercado>/testdata/`. Termo: `tomate`, salvo vitrine vazia — aí o primeiro termo com várias páginas. Sem HTTP vivo nos testes.

**Certo quando:** o fixture contém hits reais (preço, marca, unidade, paginação `hasNext`/`total` se o API tiver).

### 3. Parser TDD

Workspace TDD (um comportamento observável por ciclo, API pública). Costuras: `Parse` (cartão → `CartaoVitrine`), URL/IDs da loja default, token/header se houver, `vitrinehttp.Client` com `httptest` se o transporte for específico deste Mercado.

`CartaoCompleto`: `valor > 0`, `quantidades` ≥1, medida `g` | `ml` | `unidade`. Unidade desconhecida → `unidade` (padrão Fort). `IndicacaoPromocional` sai do cartão (selo / de-por / lista > atual).

**Certo quando:** `go test` do pacote do adapter e de `vitrinehttp` passa; nenhum teste chama o site.

### 4. Registro + seed

1. `vitrine.Defaults`: `mercado-<slug>` → `vitrinehttp.Client` (ou o client do pacote).
2. `apps/ofertas-scraper-v2/seed/mercados.json`: `{ "id": "mercado-<slug>", "nome": "<bandeira>" }`. `Coletar` ignora Mercado que não existe no catálogo.
3. Se o Mercado ainda não tem Fonte: `apps/ofertas-scraper/seed/fontes.json` — mesmo `id`/`nome` em `mercados`, Fonte `fonte-<slug>`, `tipo: "site"`, `url` = home da busca. Sem Fonte, o backoffice `testar` não acha o Mercado (ADR 0015 / 0040).
4. `apps/ofertas-scraper/seed/fontes_test.go` é conjunto fechado — incluir a Fonte nova lá.

IDs iguais nos dois seeds. Rodar `ofertas-scraper-v2 seed` (e `ofertas-scraper seed` se a Fonte entrou) antes da Coleta viva.

**Certo quando:** os dois JSON (quando couber) e o test do seed encarte batem; `Defaults` tem o Mercado; `go test ./...` em `ofertas-scraper-v2` (e `ofertas-scraper` se o seed mudou) passa.

### 5. Coleta viva até Done

`/loop` até Done. Isolar o Mercado: `POST /coletas` `{ "termo", "mercadoId" }` (o CLI `coletar` dispara **todos** os adapters). Conferir no browser (ou no JSON da mesma busca) que a vitrine esgotou e N cartões completos = N Ofertas.

`concluido` / `parcial` no mesmo dia **não voltam ao site** (ADR 0014). Depois de corrigir parser: `estado=falhou` nessa Coleta (termo+Mercado+dia) e repetir — `falhou` retenta e substitui o conjunto.

**Certo quando:** Done acima. Tick do loop que ainda está `parcial`/`falhou` ou N ≠ cartões completos → parser/paginação, não encerrar.
