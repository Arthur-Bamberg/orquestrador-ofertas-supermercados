package vitrinehttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine/fort"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrinehttp"
)

func TestClient_BuscarParseiaCorpo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("search") != "arroz" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"hits":[{"name":"Arroz Integral 1kg","brandName":"Camil","saleUnit":"UN","pricing":{"price":8.9,"promotion":false,"promotionalPrice":8.9},"quantity":{"fraction":1}}]}`))
	}))
	t.Cleanup(srv.Close)
	c := vitrinehttp.Client{
		HTTP:  srv.Client(),
		URL:   func(q string) string { return srv.URL + "?search=" + q },
		Parse: fort.ParseSearch,
	}
	got, err := c.Buscar(context.Background(), "arroz")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Valor != 8.9 {
		t.Fatalf("%#v", got)
	}
}
