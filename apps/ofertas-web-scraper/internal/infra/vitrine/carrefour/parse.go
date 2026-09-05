package carrefour

import (
	"net/url"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine/vtexsearch"
)

const BaseURL = "https://www.carrefour.com.br"

func SearchURL(query string) string {
	return BaseURL + "/api/catalog_system/pub/products/search/" + url.PathEscape(query) + "?_from=0&_to=23"
}

func ParseSearch(body []byte) ([]domain.CartaoVitrine, error) {
	return vtexsearch.ParseSearch(body)
}
