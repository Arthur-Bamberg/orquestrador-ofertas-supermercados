---
status: accepted
---

# Oferta uses quantidades[] for same-price multi-size blocks

When a flyer ties multiple discrete sizes to one price (e.g. chocolate bar 90g / 150g / 200g at R$ 5,99), that is one Oferta with `quantidades: number[]` (≥1) and a single shared `medida` — not N Ofertas and not `{quantidade, medida}` pairs. Domain dedups and sorts ascending. Extrator emits the array only for an explicit same-price list; distinct prices/blocks stay separate Ofertas; vague “diversos tamanhos” or open ranges without listed sizes are omitted. Promoção applies to the whole block. Unit price per size is a consumer expansion, not persisted rows. Contract stays unversioned (overwrite `oferta.json` / prompt); readers tolerate legacy singular `quantidade` as `[n]`. Rejected: one Oferta per size (loses the commercial block), multi-medida pairs, collapsing sizes without visual link, inventing quantities, and expanding to N records at persist time.
