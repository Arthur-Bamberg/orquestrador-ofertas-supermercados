package rissul

import (
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine/vtexsearch"
)

const BaseURL = "https://www.rissul.com.br"

func SearchURL(query string) string {
	return vtexsearch.SearchURL(BaseURL, query)
}

func ParseSearch(body []byte) ([]domain.CartaoVitrine, error) {
	return vtexsearch.ParseSearch(body)
}
