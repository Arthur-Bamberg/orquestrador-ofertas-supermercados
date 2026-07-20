---
status: accepted
---

# Vigência com dataInicio, origens na Oferta, fallbacks sempre ativos

Every Oferta has required `dataInicio` and `dataExpiracao` plus `origemDataInicio` / `origemDataExpiracao` (`extrator` | `filename` | `primeiraDescoberta` — the last only for início). Cascades (independent): início = Extrator → distinct start in filename → earliest discovery day of that filename on the Fonte (any Documento state); fim = Extrator → end in filename; if fim still empty → Falha de Extração. Filename fallbacks are always on — `fallbackDataExpiracaoFilename` is removed from Fonte (supersedes ADR 0013 and 0025). Origens are filled by application after cascades, never by the Extrator. A SET index `documento:dias:{fonteId}:{filename}` tracks discovery days for primeira descoberta.

## Considered Options

- Keep per-Fonte opt-in/auto-enable flag for expiration fallback — rejected: both dates are required and auditing belongs on the Oferta via origem fields.
- Default missing `dataInicio` only to the current discovery day — rejected: reappearance of the same filename should use the first day it was seen on that Fonte.
- Put origem only in Artefatos — rejected: validation queries need filterable fields on persisted Ofertas.
