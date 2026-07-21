# AGENTS.md — ofertas-backoffice

Backoffice local em Vite + React + TypeScript para administrar entidades do domínio de ofertas.

## Escopo

- SPA sem autenticação, consumindo a API REST em `VITE_API_BASE` (padrão `http://localhost:8080`) sob `/api/...`.
- UI administrativa funcional: tabelas, filtros, formulários, confirmações de exclusão e mensagens claras de erro da API.
- Use termos do glossário do scraper: Oferta, Fonte, Documento, Mercado, Extrator, Falha de Extração, Artefato, Uso do Extrator e Operação de Pipeline.

## Comandos

```bash
npm install
npm run dev
npm run build
```

## Regras locais

- Não introduza autenticação neste app sem decisão explícita.
- Mantenha o cliente de API tolerante a pequenas variações de payload, mas preserve os endpoints esperados em `/api`.
- Não comite `.env` nem valores secretos; versionar apenas `.env.example`.
- Prefira componentes simples e reutilizáveis antes de adicionar bibliotecas de UI.
