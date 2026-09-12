# Ofertas Backoffice

SPA local para administrar Mercados, Fontes, Documentos, Ofertas, Produtos, Marcas, Falhas de Extração, Uso do Extrator, Artefatos, Operações de Pipeline, Canal WhatsApp (Pareamento) e Conversas.

## Rodar localmente

```bash
cd apps/ofertas-backoffice
npm install
npm run dev
```

Por padrão o app usa a mesma origem: `/api` (catálogo) e `/gateway` (Canal e Conversas), via proxy do Vite ou do nginx. Copie `.env.example` para `.env` só se precisar apontar para outras bases.

```bash
cp .env.example .env
```

O Operador identifica-se em `/entrar` (nome + senha). Não há cadastro na UI — insira a linha no Postgres (veja o `.env.example` da raiz). A API do backoffice deve estar rodando em `:8080` para o proxy local padrão.

No monorepo: `docker compose up --build` sobe API + SPA (ADR 0008). O browser usa `http://localhost:5173`.

## Build

```bash
npm run build
```
