# Ofertas Backoffice

SPA local para administrar Mercados, Fontes, Documentos, Ofertas, Produtos, Marcas, Falhas de Extração, Uso do Extrator, Artefatos e Operações de Pipeline.

## Rodar localmente

```bash
cd apps/ofertas-backoffice
npm install
npm run dev
```

Por padrão o app usa `VITE_API_BASE=http://localhost:8080` e chama endpoints REST em `/api/...`. Copie `.env.example` para `.env` se precisar apontar para outra API.

```bash
cp .env.example .env
```

A API do backoffice deve estar rodando em `:8080` para a experiência local padrão.

## Build

```bash
npm run build
```
