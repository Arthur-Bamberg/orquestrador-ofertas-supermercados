package osuper

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

const PageSize = 12 // Fort/Osuper rejects size=24 (returns unfiltered catalog)

func SearchURL(account, storeID, query string, from, size int) string {
	q := url.Values{}
	q.Set("search", query)
	q.Set("size", strconv.Itoa(size))
	q.Set("from", strconv.Itoa(from))
	return "https://sense.osuper.com.br/" + account + "/" + storeID + "/search?" + q.Encode()
}

type searchDoc struct {
	Hits     []hit `json:"hits"`
	Total    int   `json:"total"`
	NextFrom int   `json:"nextFrom"`
	HasNext  bool  `json:"hasNext"`
}

type hit struct {
	Name      string `json:"name"`
	BrandName string `json:"brandName"`
	SaleUnit  string `json:"saleUnit"`
	Pricing   struct {
		Price            float64 `json:"price"`
		Promotion        bool    `json:"promotion"`
		PromotionalPrice float64 `json:"promotionalPrice"`
	} `json:"pricing"`
	Quantity struct {
		Fraction float64 `json:"fraction"`
	} `json:"quantity"`
}

func ParseSearch(body []byte) (domain.PaginaVitrine, error) {
	var doc searchDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return domain.PaginaVitrine{}, err
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
		medida, qty := MedidaDoCartao(h.SaleUnit, qty)
		out = append(out, domain.CartaoVitrine{
			Nome:                 h.Name,
			Marca:                h.BrandName,
			Valor:                valor,
			Quantidades:          []float64{qty},
			Medida:               medida,
			IndicacaoPromocional: h.Pricing.Promotion || (h.Pricing.PromotionalPrice > 0 && h.Pricing.PromotionalPrice < h.Pricing.Price),
		})
	}
	return domain.PaginaVitrine{
		Cartoes:  out,
		Total:    doc.Total,
		NextFrom: doc.NextFrom,
		HasNext:  doc.HasNext,
	}, nil
}

// MedidaDoCartao maps Fort/Osuper saleUnit to catalog Medida; unknown → unidade.
// KG/L are price-per-unit measures on the shelf (1 kg / 1 L), not the min purchase fraction.
func MedidaDoCartao(saleUnit string, fraction float64) (store.Medida, float64) {
	u := strings.ToUpper(strings.TrimSpace(saleUnit))
	switch u {
	case "KG":
		return store.MedidaG, 1000
	case "G", "GR":
		return store.MedidaG, fraction
	case "L", "LT":
		return store.MedidaML, 1000
	case "ML":
		return store.MedidaML, fraction
	default:
		return store.MedidaUnidade, fraction
	}
}
