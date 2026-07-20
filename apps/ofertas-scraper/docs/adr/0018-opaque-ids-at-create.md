# Opaque IDs assigned at create time in application

Mercado, Fonte, Produto, Marca, Documento, and Oferta persist with opaque string IDs (UUID or ULID) minted when the application creates the entity. Natural keys remain for lookup only: Documento by Fonte + filename + discovery day; Produto and Marca by normalized name. IDs are not derived from those keys (no hash-as-id), so renames, collisions, and future identity rules stay decoupled from foreign keys on Oferta. Seeded Mercado/Fonte also get opaque IDs at creation time.
