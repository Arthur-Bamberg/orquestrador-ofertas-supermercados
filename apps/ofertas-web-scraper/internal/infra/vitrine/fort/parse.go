package fort

import (
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine/osuper"
)

const (
	BaseURL   = "https://www.fortatacadista.com.br"
	AccountID = "334"
	StoreID   = "1687" // Canoas
)

func SearchURL(query string) string {
	return osuper.SearchURL(AccountID, StoreID, query)
}

func ParseSearch(body []byte) ([]domain.CartaoVitrine, error) {
	return osuper.ParseSearch(body)
}
