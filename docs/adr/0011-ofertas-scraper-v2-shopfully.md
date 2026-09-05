# ofertas-scraper-v2: Shopfully image job as its own app

A second scraper lives in `apps/ofertas-scraper-v2` because the Fonte is different (one Shopfully listing with many Mercados, page JPEGs instead of PDF→raster). It is a one-shot CLI like v1, registered in `go.work`, with its own module and Dockerfile. Shared catalog write and Extrator stay out until a later slice. Rejected: folding Shopfully into v1 FonteClient, copying v1 `internal/`.
