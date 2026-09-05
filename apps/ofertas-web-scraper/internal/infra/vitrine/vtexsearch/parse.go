package vtexsearch

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func SearchURL(base, query string) string {
	return strings.TrimRight(base, "/") + "/api/catalog_system/pub/products/search?ft=" + url.QueryEscape(query) + "&_from=0&_to=23"
}

type product struct {
	ProductName string `json:"productName"`
	Brand       string `json:"brand"`
	Items       []struct {
		MeasurementUnit string  `json:"measurementUnit"`
		UnitMultiplier  float64 `json:"unitMultiplier"`
		Sellers         []struct {
			Offer struct {
				Price     float64 `json:"Price"`
				ListPrice float64 `json:"ListPrice"`
			} `json:"commertialOffer"`
		} `json:"sellers"`
	} `json:"items"`
}

func ParseSearch(body []byte) ([]domain.CartaoVitrine, error) {
	var products []product
	if err := json.Unmarshal(body, &products); err != nil {
		return nil, err
	}
	out := make([]domain.CartaoVitrine, 0, len(products))
	for _, p := range products {
		cartao := domain.CartaoVitrine{
			Nome:        p.ProductName,
			Marca:       marcaCatalogo(p.Brand),
			Medida:      store.MedidaUnidade,
			Quantidades: []float64{1},
		}
		if len(p.Items) > 0 {
			item := p.Items[0]
			cartao.Medida, cartao.Quantidades = medidaEQtd(item.MeasurementUnit, item.UnitMultiplier)
			if len(item.Sellers) > 0 {
				offer := item.Sellers[0].Offer
				cartao.Valor = offer.Price
				cartao.IndicacaoPromocional = offer.ListPrice > offer.Price
			}
		}
		out = append(out, cartao)
	}
	return out, nil
}

func marcaCatalogo(brand string) string {
	b := strings.TrimSpace(brand)
	if strings.EqualFold(b, "Não Informado") || strings.EqualFold(b, "Nao Informado") {
		return ""
	}
	return b
}

func medidaEQtd(unit string, multiplier float64) (store.Medida, []float64) {
	if multiplier <= 0 {
		multiplier = 1
	}
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "kg":
		return store.MedidaG, []float64{multiplier * 1000}
	case "g":
		return store.MedidaG, []float64{multiplier}
	case "l", "lt":
		return store.MedidaML, []float64{multiplier * 1000}
	case "ml":
		return store.MedidaML, []float64{multiplier}
	default:
		return store.MedidaUnidade, []float64{multiplier}
	}
}
