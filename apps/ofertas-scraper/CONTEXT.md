# Ofertas Scraper

Sistema que coleta PDFs de Fontes de ofertas, rasteriza páginas em imagens, extrai Ofertas via Extrator e persiste Produtos, Mercados e histórico de preços com Artefatos para debug.

## Language

**Oferta**:
Observação de preço extraída de um Documento: valor (preço de prateleira / unitário sem condição — o preço “cheio” do bloco), `quantidades[]` (≥1, mesma Medida), `dataInicio` e `dataExpiracao` (vigência no encarte; `dataInicio` ≤ `dataExpiracao`; a expiração pode estar no passado para histórico; ambas podem ser futuras) e promoção opcional; refere-se a um Produto em um Mercado, com Marca opcional. Vários tamanhos no mesmo preço do encarte (lista discreta explícita, ex. 90g / 150g / 200g) são uma só Oferta; preços ou blocos distintos geram Ofertas distintas; intervalo vago ou “diversos tamanhos” sem números não vira Oferta. Domain normaliza `quantidades` (dedup + ordem crescente). Promoção, quando houver, aplica ao bloco inteiro. Preço unitário por tamanho é expansão do consumidor da base, não linhas extras persistidas. Cada data carrega origem (`origemDataInicio` / `origemDataExpiracao`) com valor `extrator`, `filename` ou — só no início — `primeiraDescoberta`. Cascata de `dataInicio`: Extrator → início distinto no nome (incl. intervalo no filename; token ISO único não conta como início) → dia da primeira descoberta na Fonte (menor dia de qualquer Documento com aquele nome na Fonte, independente do estado). Cascata de `dataExpiracao`: Extrator → fim no nome (intervalo ou token ISO; sempre tentada; sem flag na Fonte); se ambas falharem, Falha de Extração — sem inventar fim. As duas cascatas são independentes. Toda Oferta tem `dataInicio`, `dataExpiracao` e as duas origens — não há forma legítima sem esses campos. O histórico de preços de um Produto é o conjunto de Ofertas ao longo do tempo (via Documentos); o “período atual” é filtro do consumidor da base, não um estado embutido no Produto.
_Avoid_: Deal, item, listing, produto, histórico de produto (como entidade separada), dataComeço, vigenciaInicio, quantidade (singular como campo do contrato)

**Produto**:
Identidade de catálogo do que está à venda, sem marca (ex.: “Arroz integral”, “Arroz branco parboilizado”); carrega categorias taxonômicas para navegação e agregação. Distinta por tipo vendável; N Marcas aparecem via Ofertas, não como lista fixa no Produto.
_Avoid_: Oferta, item, SKU, variante (como substituto de Produto), marca

**Marca**:
Identidade comercial do fabricante ou rótulo (ex.: Camil, Tio João), independente do Produto e do Mercado. Opcional na Oferta quando o encarte não traz marca. Quando presente, o Extrator a identifica nas imagens e o domain casa ou cria a Marca.
_Avoid_: fabricante como texto solto na Oferta, brand

**Categoria**:
Rótulo taxonômico de um Produto para filtrar e agrupar (ex.: mercearia, grãos, arroz). Não distingue tipos vendáveis — isso são Produtos diferentes. Novas categorias vistas na extração unem-se às já existentes do Produto.
_Avoid_: tipo, variante, tag solta na Oferta

**Fonte**:
URL configurada e persistida cuja resposta revela URLs de PDFs (HTML/JS/JSON); o nome do arquivo identifica o Documento e o download usa a URL do link. Pode apontar para página HTML ou endpoint JSON de ofertas. Pertence a um Mercado; pode incluir filtro opcional por regex sobre o nome do Documento. Fallbacks de vigência (filename / primeira descoberta) são sempre ativos — não há flag por Fonte. Se o Extrator envia a data, ela prevalece.
_Avoid_: Site, link, URL, origem (como sinônimo de Fonte)

**Mercado**:
Identidade comercial (rede ou bandeira) à qual uma Fonte pertence; sujeito da comparação de Ofertas e do histórico de preços entre estabelecimentos. Não é extraído das imagens — vem da configuração da Fonte.
_Avoid_: loja, supermercado, site, Fonte

**Documento**:
PDF identificado em uma Fonte pelo nome do arquivo e pelo dia da descoberta; rastreado ao longo do processamento (download, rasterização, extração). Estados persistidos: processando, concluído, parcial, falhou. Entra em processando no início do tratamento (não há estado persistido “listado mas ainda não baixado”). Sem pelo menos uma Oferta persistida (lista vazia do Extrator ou só Falhas de Extração), o Documento termina em falhou. Em falha dura de processamento (incluindo Extrator indisponível após retentativas), permanece consultável com o motivo do último erro.
_Avoid_: PDF, arquivo, anexo, descoberto (como estado persistido)

**Extrator**:
Capacidade de obter candidatos a Oferta a partir das imagens de um Documento (rótulos de produto, marca e categorias, mais preço, `dataInicio`, `dataExpiracao` e promoção). Espera-se as duas datas em todo candidato; ausências são preenchidas pelos fallbacks de vigência (a origem fica na Oferta, não na saída do Extrator).
_Avoid_: Gemini, IA, conversor, parser, LLM

**Medida**:
Unidade compartilhada por todas as `quantidades` de uma Oferta, sempre normalizada para `g`, `ml` ou `unidade` (kg → 1000 g; L → 1000 ml).
_Avoid_: unidade de medida, kg, litro, L

**Promoção**:
Condição comercial opcional de uma Oferta: sempre com `valorPromocional` (preço já sob a condição — unitário ou do pack, conforme o encarte mostra; não o total do leve/pague salvo se for o único número exibido), mais canal e/ou mecânica de quantidade. Canal (no máximo um): preço exclusivo de cartão (`promocaoCartao`) ou de clube/fidelidade/app (`promocaoClube`) — distintos; o encarte indica pelo termo (cartão/card → cartão; clube sem cartão → clube). Mecânica (no máximo uma): leve/pague (“leve X pague Y”, “LV16 PG15”) ou quantidade com valor promocional (`quantidadePromocao`, ex.: “a partir de N un.”). “Leve N” / pack multiunidade com preço de kit e nota unitária informativa não é Promoção — é o `valor` da Oferta do pack. Exige pelo menos canal ou mecânica; canal e mecânica podem coexistir quando descrevem a **mesma** condição (mesmo `valorPromocional`, ex.: leve/pague no preço do clube). Se o bloco só mostra um preço com selo de canal, `valor` da Oferta e `valorPromocional` são iguais e o canal fica preenchido. Se o mesmo bloco tiver duas condições com preços promocionais distintos (ex.: “a partir de N” e cartão), não compor nem criar segunda Oferta — prevalece o canal (cartão/clube) e seu preço.
_Avoid_: desconto, oferta especial, deal, vínculo, canal genérico (como substituto de cartão/clube)

**Falha de Extração**:
Registro de uma tentativa de Oferta que não passou na validação, vinculada ao Documento de origem: código estável do motivo, detalhe livre opcional e cópia do candidato rejeitado.
_Avoid_: export com erro, erro de IA, rejeição

**Artefato**:
Material obtido ou gerado em uma tentativa de processamento de um Documento e retido para debug — tipicamente PDF original, imagens enviadas ao Extrator, resposta bruta do Extrator, Uso do Extrator e resultado validado (Ofertas e Falhas de Extração). Em falha dura, persiste-se só o que a tentativa chegou a produzir (best-effort); não se fabricam placeholders para etapas que não rodaram. Cada reprocessamento acrescenta uma nova tentativa; tentativas anteriores permanecem. Ofertas e Falhas de Extração persistidas no estado atual do Documento são substituídas na nova tentativa — o histórico de tentativas vive nos Artefatos. Hoje em armazenamento local; depois em bucket.
_Avoid_: arquivo, blob, export, attachment, log

**Uso do Extrator**:
Registro de consumo de tokens de uma tentativa de extração de um Documento: path do Artefato da tentativa, totais de tokens de input (prompt), cache e output, com detalhe por página quando houver várias chamadas. Persiste no Redis para consulta e espelha-se no Artefato da mesma tentativa. Não substitui o conteúdo extraído — só mensura o custo da chamada ao Extrator.
_Avoid_: billing, fatura, métrica genérica, log de API

## Qualidade local

Antes de cada commit, o hook do monorepo (raiz) executa `go test ./...` em cada módulo do `go.work`. Instalar uma vez por clone com `../../scripts/install-git-hooks.sh`. Detalhes em [`AGENTS.md`](./AGENTS.md) e ADR de workspace 0003.
