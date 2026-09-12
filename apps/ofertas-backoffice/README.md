# Ofertas Backoffice

SPA local para administrar Mercados, Fontes, Documentos, Ofertas, Produtos, Marcas, Falhas de Extração, Uso do Extrator, Artefatos, Operações de Pipeline, Canal WhatsApp (Pareamento) e Conversas.

## Rodar localmente

```bash
cd apps/ofertas-backoffice
npm install
npm run dev
```

Por padrão o app usa `VITE_API_BASE=http://localhost:8080` (catálogo) e `VITE_GATEWAY_BASE=http://localhost:8090` (Canal e Conversas). Copie `.env.example` para `.env` se precisar apontar para outras bases ou definir `VITE_GATEWAY_TOKEN` (Pareamento, envio).

```bash
cp .env.example .env
```

A API do backoffice deve estar rodando em `:8080` para a experiência local padrão.

No monorepo: `docker compose up --build` sobe API + SPA (ADR 0008). O browser usa `VITE_API_BASE` (padrão `http://localhost:8080`).

## Build

```bash
npm run build
```
