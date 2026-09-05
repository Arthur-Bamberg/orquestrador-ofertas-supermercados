package vitrine

import (
	"net/http"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine/asun"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine/carrefour"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine/fort"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine/rissul"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrinehttp"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

const (
	MercadoFort      store.MercadoID = "mercado-fort"
	MercadoCarrefour store.MercadoID = "mercado-carrefour"
	MercadoAsun      store.MercadoID = "mercado-asun"
	MercadoRissul    store.MercadoID = "mercado-rissul"
)

func Defaults(httpClient *http.Client) map[store.MercadoID]domain.Vitrine {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return map[store.MercadoID]domain.Vitrine{
		MercadoFort:      vitrinehttp.Client{HTTP: httpClient, URL: fort.SearchURL, Parse: fort.ParseSearch},
		MercadoCarrefour: vitrinehttp.Client{HTTP: httpClient, URL: carrefour.SearchURL, Parse: carrefour.ParseSearch},
		MercadoAsun:      vitrinehttp.Client{HTTP: httpClient, URL: asun.SearchURL, Parse: asun.ParseSearch},
		MercadoRissul:    vitrinehttp.Client{HTTP: httpClient, URL: rissul.SearchURL, Parse: rissul.ParseSearch},
	}
}
