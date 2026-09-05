# Shopfully listing to page JPEGs for Mercados we already have

v1 discovers PDFs per Fonte. Shopfully publishes many supermarket Encartes on one listing, already as page images. v2 GETs that listing, keeps Encartes whose Mercado name matches the seed (Fort Atacadista, Stok Center, Via Atacadista), follows the viewer to the Zmags publication API, and downloads only the highest-resolution page JPEGs. Rejected: Playwright, downloading logos/covers, rasterizing PDFs, calling the Extrator in this slice.
