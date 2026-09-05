package carrefour

import (
	"encoding/json"
	"net/url"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

const BaseURL = "https://www.carrefour.com.br"

func SearchURL(query string) string {
	return BaseURL + "/api/catalog_system/pub/products/search?ft=" + url.QueryEscape(query)
}

type vtexProduct struct {
	ProductName string `json:"productName"`
	Brand       string `json:"brand"`
	Items       []struct {
		Sellers []struct {
			Offer struct {
				Price     float64 `json:"Price"`
				ListPrice float64 `json:"ListPrice"`
			} `json:"commertialOffer"`
		} `json:"sellers"`
	} `json:"items"`
}

func ParseSearch(body []byte) ([]domain.CartaoVitrine, error) {
	var products []vtexProduct
	if err := json.Unmarshal(body, &products); err != nil {
		return nil, err
	}
	out := make([]domain.CartaoVitrine, 0, len(products))
	for _, p := range products {
		cartao := domain.CartaoVitrine{Nome: p.ProductName, Marca: p.Brand, Medida: store.MedidaUnidade, Quantidades: []float64{1}}
		if len(p.Items) > 0 && len(p.Items[0].Sellers) > 0 {
			offer := p.Items[0].Sellers[0].Offer
			cartao.Valor = offer.Price
			cartao.IndicacaoPromocional = offer.ListPrice > offer.Price
		}
		out = append(out, cartao)
	}
	return out, nil
}
