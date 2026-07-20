# Past dataExpiracao is valid for price history

Domain accepts `dataExpiracao` earlier than the Documento discovery day. The date is a fact from the flyer (or filename fallback), not a “still on sale” filter. Expired Ofertas remain first-class observations so the catalog can serve price history across Mercados. Filtering to currently valid offers belongs to query/read models, not extraction validation.
