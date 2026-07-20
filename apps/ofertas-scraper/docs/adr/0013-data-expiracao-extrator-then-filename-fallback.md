---
status: superseded by ADR-0028
---

# dataExpiracao prefers Extrator; optional filename fallback per Fonte

`dataExpiracao` on an Oferta comes from the Extrator when present. If missing, and the Fonte has filename-expiration fallback enabled (`fallbackDataExpiracaoFilename`, default `false` — opt-in per Fonte), domain may parse a date from the Documento filename via a dedicated port and use it. If the Extrator value is present, it always wins — no conflict Falha against the filename. If the Extrator omits the date and fallback is disabled or parse fails, the candidate is a Falha de Extração. The flag lives on the Fonte, not in the Extrator schema.
