module github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/cmd/merge-produtos

go 1.25.0

require (
	github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper v0.0.0
	github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store v0.0.0
	github.com/joho/godotenv v1.5.1
)

replace github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper v0.0.0 => ../..
replace github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store v0.0.0 => ../../../modules/ofertas-store
