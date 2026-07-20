---
status: superseded by ADR-0028
---

# dataExpiracao required; auto-enable filename fallback on Fonte

The Extrator contract requires `dataExpiracao` on every candidate (schema keeps the field required) so the model is pushed to read the flyer date instead of omitting it. When a candidate still arrives with missing or empty `dataExpiracao`, the job persists `fallbackDataExpiracaoFilename = true` on that Documento’s Fonte and applies filename parsing in the same attempt. Extrator-provided dates still win over the filename. If fallback is on (or just enabled) and parse fails, the candidate remains a Falha de Extração. The flag stays on the Fonte, never on the Mercado.

## Considered Options

- Keep `dataExpiracao` optional in the schema and rely on manual opt-in fallback (ADR 0013) — rejected: optional structured output invites omission and leaves Fontes that need filename dates permanently under-configured.
- Require the field and drop filename fallback — rejected: real Fontes encode validity in the PDF name and the Extrator still occasionally returns empty dates.
