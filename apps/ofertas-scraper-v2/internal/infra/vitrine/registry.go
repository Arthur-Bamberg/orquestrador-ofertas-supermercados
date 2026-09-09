package vitrine

import (
	"net/http"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrine/fort"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrine/osuper"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrinehttp"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

const MercadoFort store.MercadoID = "mercado-fort"

func Defaults(httpClient *http.Client) map[store.MercadoID]domain.Vitrine {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	tokens := &fort.TokenProvider{HTTP: httpClient}
	return map[store.MercadoID]domain.Vitrine{
		MercadoFort: vitrinehttp.Client{
			HTTP:     httpClient,
			URL:      fort.SearchURL,
			Parse:    fort.ParseSearch,
			PageSize: osuper.PageSize,
			Headers:  tokens.Header,
		},
	}
}
