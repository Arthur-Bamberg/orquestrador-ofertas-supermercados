package fort

import (
	"encoding/json"
	"net/url"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

const BaseURL = "https://www.fortatacadista.com.br"

func SearchURL(query string) string {
	return BaseURL + "/api/catalog_system/pub/products/search?ft=" + url.QueryEscape(query)
}

type searchDoc struct {
	Products []struct {
		Name      string  `json:"name"`
		Brand     string  `json:"brand"`
		Price     float64 `json:"price"`
		ListPrice float64 `json:"listPrice"`
		Quantity  float64 `json:"quantity"`
		Unit      string  `json:"unit"`
	} `json:"products"`
}

func ParseSearch(body []byte) ([]domain.CartaoVitrine, error) {
	var doc searchDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, err
	}
	out := make([]domain.CartaoVitrine, 0, len(doc.Products))
	for _, p := range doc.Products {
		out = append(out, domain.CartaoVitrine{
			Nome:                 p.Name,
			Marca:                p.Brand,
			Valor:                p.Price,
			Quantidades:          []float64{p.Quantity},
			Medida:               store.Medida(p.Unit),
			IndicacaoPromocional: p.ListPrice > p.Price,
		})
	}
	return out, nil
}
