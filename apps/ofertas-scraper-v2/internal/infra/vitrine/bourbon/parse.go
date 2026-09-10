package bourbon

import (
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrine/osuper"
)

const (
	BaseURL  = "https://www.zaffari.com.br"
	PageSize = 12
)

func SearchURL(query string, from, size int) string {
	if size <= 0 {
		size = PageSize
	}
	page := from/size + 1
	q := url.Values{}
	q.Set("query", query)
	q.Set("page", strconv.Itoa(page))
	q.Set("count", strconv.Itoa(size))
	q.Set("hideUnavailableItems", "true")
	return BaseURL + "/api/io/_v/api/intelligent-search/product_search/?" + q.Encode()
}

type searchDoc struct {
	Products        []product  `json:"products"`
	RecordsFiltered int        `json:"recordsFiltered"`
	Pagination      pagination `json:"pagination"`
}

type pagination struct {
	Current pageRef `json:"current"`
	Next    pageRef `json:"next"`
	PerPage int     `json:"perPage"`
}

type pageRef struct {
	Index int `json:"index"`
}

type product struct {
	ProductName string `json:"productName"`
	Brand       string `json:"brand"`
	Items       []item `json:"items"`
}

type item struct {
	MeasurementUnit string   `json:"measurementUnit"`
	UnitMultiplier  float64  `json:"unitMultiplier"`
	Sellers         []seller `json:"sellers"`
}

type seller struct {
	CommertialOffer struct {
		Price     float64 `json:"Price"`
		ListPrice float64 `json:"ListPrice"`
	} `json:"commertialOffer"`
}

func ParseSearch(body []byte) (domain.PaginaVitrine, error) {
	var doc searchDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return domain.PaginaVitrine{}, err
	}
	out := make([]domain.CartaoVitrine, 0, len(doc.Products))
	for _, p := range doc.Products {
		if len(p.Items) == 0 || len(p.Items[0].Sellers) == 0 {
			continue
		}
		it := p.Items[0]
		offer := it.Sellers[0].CommertialOffer
		frac := it.UnitMultiplier
		if frac <= 0 {
			frac = 1
		}
		medida, qty := osuper.MedidaDoCartao(it.MeasurementUnit, frac)
		out = append(out, domain.CartaoVitrine{
			Nome:                 p.ProductName,
			Marca:                p.Brand,
			Valor:                offer.Price,
			Quantidades:          []float64{qty},
			Medida:               medida,
			IndicacaoPromocional: offer.ListPrice > offer.Price && offer.Price > 0,
		})
	}
	pageSize := doc.Pagination.PerPage
	if pageSize <= 0 {
		pageSize = PageSize
	}
	return domain.PaginaVitrine{
		Cartoes:  out,
		Total:    doc.RecordsFiltered,
		NextFrom: doc.Pagination.Current.Index * pageSize,
		HasNext:  doc.Pagination.Next.Index > doc.Pagination.Current.Index,
	}, nil
}
