# Ofertas Scraper v2 (Shopfully)

Coleta imagens de páginas de Encartes a partir da listagem Shopfully de supermercados em Canoas. Não baixa PDF, logo nem capa da grade — só as JPEGs de página do viewer, e só dos Mercados que já temos no catálogo.

## Language

**Encarte**:
Folheto digital listado na Fonte Shopfully, identificado pelo `flyerId`, com um Mercado (nome da rede no card) e um viewer. Vários Encartes da mesma listagem podem ser de Mercados diferentes.
_Avoid_: flyer, volantino, PDF, Documento (como identidade deste app)

**Página**:
Uma imagem JPEG de uma folha do Encarte, na maior resolução publicada pelo viewer. É o que este job baixa.
_Avoid_: thumbnail, capa da grade, logo, artefato genérico

**Mercado**:
Identidade comercial já conhecida no catálogo (seed deste app, os mesmos nomes do scraper v1). O job ignora Encartes cujo nome no card não casa com um Mercado que temos.
_Avoid_: loja, retailer, todos os cards da listagem

**Fonte**:
A URL de listagem Shopfully (`/canoas/supermercados`). Uma Fonte agrega Encartes de vários Mercados.
_Avoid_: site, API Zmags (é infra do viewer)

Glossário compartilhado de Oferta/Produto/Extrator: [`../ofertas-scraper/CONTEXT.md`](../ofertas-scraper/CONTEXT.md). Este app **não** chama o Extrator nem persiste Oferta.
