package asun

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

const BaseURL = "https://www.asun.com.br"

func SearchURL(query string) string {
	return BaseURL + "/busca?q=" + url.QueryEscape(query)
}

var cardRE = regexp.MustCompile(`(?s)<article class="product-card" data-promo="([01])">.*?<h2>([^<]+)</h2>.*?<span class="brand">([^<]+)</span>.*?<span class="price">([^<]+)</span>.*?<span class="qty">([^<]+)</span>.*?<span class="unit">([^<]+)</span>`)

func ParseSearch(body []byte) ([]domain.CartaoVitrine, error) {
	matches := cardRE.FindAllSubmatch(body, -1)
	out := make([]domain.CartaoVitrine, 0, len(matches))
	for _, m := range matches {
		price, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(string(m[4])), ",", "."), 64)
		if err != nil {
			continue
		}
		qty, err := strconv.ParseFloat(strings.TrimSpace(string(m[5])), 64)
		if err != nil {
			continue
		}
		out = append(out, domain.CartaoVitrine{
			Nome:                 strings.TrimSpace(string(m[2])),
			Marca:                strings.TrimSpace(string(m[3])),
			Valor:                price,
			Quantidades:          []float64{qty},
			Medida:               store.Medida(strings.TrimSpace(string(m[6]))),
			IndicacaoPromocional: string(m[1]) == "1",
		})
	}
	return out, nil
}
