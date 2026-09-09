# Fonte carries tipo encarte | site

Collection kind lives on **Fonte**, not Documento: the seed configures Fontes (Fort, Stok, Via), and Documento is always a PDF from an encarte Fonte. `encarte` (default) is the PDF pipeline; `site` stays in the catalog for Coleta and is skipped by the daily job and on-demand discover. Rejected: `tipo` on Documento (would not be seedable and would mix Coleta into Documento); using `ativa=false` for Fort/Stok (that flag means paused, not “this Mercado is collected as vitrine”).
