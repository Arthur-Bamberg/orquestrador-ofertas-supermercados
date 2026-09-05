package osuper

import (
	"encoding/json"
	"net/url"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func SearchURL(account, storeID, query string) string {
	q := url.Values{}
	q.Set("search", query)
	q.Set("size", "24")
	q.Set("from", "0")
	return "https://sense.osuper.com.br/" + account + "/" + storeID + "/search?" + q.Encode()
}

type searchDoc struct {
	Hits []hit `json:"hits"`
}

type hit struct {
	Name      string `json:"name"`
	BrandName string `json:"brandName"`
	Pricing   struct {
		Price            float64 `json:"price"`
		Promotion        bool    `json:"promotion"`
		PromotionalPrice float64 `json:"promotionalPrice"`
	} `json:"pricing"`
	Quantity struct {
		Fraction float64 `json:"fraction"`
	} `json:"quantity"`
}

func ParseSearch(body []byte) ([]domain.CartaoVitrine, error) {
	var doc searchDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, err
	}
	out := make([]domain.CartaoVitrine, 0, len(doc.Hits))
	for _, h := range doc.Hits {
		valor := h.Pricing.Price
		if h.Pricing.Promotion && h.Pricing.PromotionalPrice > 0 {
			valor = h.Pricing.PromotionalPrice
		}
		qty := h.Quantity.Fraction
		if qty <= 0 {
			qty = 1
		}
		out = append(out, domain.CartaoVitrine{
			Nome:                 h.Name,
			Marca:                h.BrandName,
			Valor:                valor,
			Quantidades:          []float64{qty},
			Medida:               store.MedidaUnidade,
			IndicacaoPromocional: h.Pricing.Promotion || (h.Pricing.PromotionalPrice > 0 && h.Pricing.PromotionalPrice < h.Pricing.Price),
		})
	}
	return out, nil
}
