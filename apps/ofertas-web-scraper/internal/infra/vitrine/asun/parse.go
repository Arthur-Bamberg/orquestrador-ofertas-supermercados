package asun

import (
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine/osuper"
)

const (
	BaseURL   = "https://www.asunonline.com.br"
	AccountID = "319"
	StoreID   = "1540" // Cavalhada
)

func SearchURL(query string) string {
	return osuper.SearchURL(AccountID, StoreID, query)
}

func ParseSearch(body []byte) ([]domain.CartaoVitrine, error) {
	return osuper.ParseSearch(body)
}
