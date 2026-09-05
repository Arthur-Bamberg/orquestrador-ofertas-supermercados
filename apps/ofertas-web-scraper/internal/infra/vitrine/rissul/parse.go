package rissul

import (
	"encoding/json"
	"net/url"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

const BaseURL = "https://www.rissul.com.br"

func SearchURL(query string) string {
	return BaseURL + "/api/busca?q=" + url.QueryEscape(query)
}

type searchDoc struct {
	Items []struct {
		Title      string   `json:"title"`
		Marca      string   `json:"marca"`
		PrecoAtual float64  `json:"precoAtual"`
		PrecoDe    *float64 `json:"precoDe"`
		Quantidade float64  `json:"quantidade"`
		Medida     string   `json:"medida"`
	} `json:"items"`
}

func ParseSearch(body []byte) ([]domain.CartaoVitrine, error) {
	var doc searchDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, err
	}
	out := make([]domain.CartaoVitrine, 0, len(doc.Items))
	for _, it := range doc.Items {
		promo := it.PrecoDe != nil && *it.PrecoDe > it.PrecoAtual
		out = append(out, domain.CartaoVitrine{
			Nome:                 it.Title,
			Marca:                it.Marca,
			Valor:                it.PrecoAtual,
			Quantidades:          []float64{it.Quantidade},
			Medida:               store.Medida(it.Medida),
			IndicacaoPromocional: promo,
		})
	}
	return out, nil
}
