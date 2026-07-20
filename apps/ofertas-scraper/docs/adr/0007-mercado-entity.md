# Mercado is an entity; Fonte and Oferta reference it by id

Mercado is a first-class persisted identity (retail banner/chain). Each Fonte belongs to a Mercado via `mercadoId`; Ofertas also carry `mercadoId` so price history and cross-market comparison do not depend on joining through Fonte alone. The Extrator never emits Mercado — it comes only from Fonte configuration when the Documento is processed. Rejected alternatives: treating Mercado as a display string on Fonte only, or denormalizing only a free-text name onto Oferta without a stable id.
