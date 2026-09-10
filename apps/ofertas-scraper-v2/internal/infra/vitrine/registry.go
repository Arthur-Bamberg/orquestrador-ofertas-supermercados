package vitrine

import (
	"net/http"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrine/bourbon"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrine/carrefour"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrine/fort"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrine/osuper"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrine/stok"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrinehttp"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

const MercadoFort store.MercadoID = "mercado-fort"
const MercadoStok store.MercadoID = "mercado-stok"
const MercadoBourbon store.MercadoID = "mercado-bourbon"
const MercadoCarrefour store.MercadoID = "mercado-carrefour"

func Defaults(httpClient *http.Client) map[store.MercadoID]domain.Vitrine {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	fortTokens := &fort.TokenProvider{HTTP: httpClient}
	stokTokens := &stok.TokenProvider{HTTP: httpClient}
	return map[store.MercadoID]domain.Vitrine{
		MercadoFort: vitrinehttp.Client{
			HTTP:     httpClient,
			URL:      fort.SearchURL,
			Parse:    fort.ParseSearch,
			PageSize: osuper.PageSize,
			Headers:  fortTokens.Header,
		},
		MercadoStok: vitrinehttp.Client{
			HTTP:     httpClient,
			URL:      stok.SearchURL,
			Parse:    stok.ParseSearch,
			PageSize: stok.PageSize,
			Headers:  stokTokens.Header,
		},
		MercadoBourbon: vitrinehttp.Client{
			HTTP:     httpClient,
			URL:      bourbon.SearchURL,
			Parse:    bourbon.ParseSearch,
			PageSize: bourbon.PageSize,
		},
		MercadoCarrefour: vitrinehttp.Client{
			HTTP:     httpClient,
			URL:      carrefour.SearchURL,
			Parse:    carrefour.ParseSearch,
			PageSize: carrefour.PageSize,
		},
	}
}
