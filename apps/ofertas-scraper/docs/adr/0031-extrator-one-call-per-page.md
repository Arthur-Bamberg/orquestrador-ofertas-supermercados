# Extrator: one Gemini call per page

Dense encartes lose Ofertas when all page images go in a single vision call (model samples instead of exhausting the grid). The Gemini Extrator adapter now issues one `generateContent` per page image and merges `ofertas` into one raw JSON for Artefatos. Rejected for now: tiling within a page (more cost/complexity) and raising resolution alone without partitioning.
