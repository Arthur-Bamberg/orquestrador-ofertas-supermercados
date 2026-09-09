package vitrinehttp_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrine/fort"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrinehttp"
)

func TestClient_BuscarParseiaCorpo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("search") != "tomate" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		if r.Header.Get(fort.SearchTokenHeader) != "tok" {
			t.Fatalf("token header=%q", r.Header.Get(fort.SearchTokenHeader))
		}
		_, _ = w.Write([]byte(`{"hits":[{"name":"Tomate Carmem 1kg","brandName":"Carmem","saleUnit":"UN","pricing":{"price":6.9,"promotion":false,"promotionalPrice":6.9},"quantity":{"fraction":1}}],"total":1,"hasNext":false,"nextFrom":1}`))
	}))
	t.Cleanup(srv.Close)
	c := vitrinehttp.Client{
		HTTP: srv.Client(),
		URL: func(q string, from, size int) string {
			return fmt.Sprintf("%s?search=%s&from=%d&size=%d", srv.URL, q, from, size)
		},
		Parse: fort.ParseSearch,
		Headers: func(context.Context) (http.Header, error) {
			h := make(http.Header)
			h.Set(fort.SearchTokenHeader, "tok")
			return h, nil
		},
	}
	got, err := c.Buscar(context.Background(), "tomate")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Esgotada || len(got.Cartoes) != 1 || got.Cartoes[0].Valor != 6.9 {
		t.Fatalf("%#v", got)
	}
}

func TestClient_BuscarPaginaAteEsgotar(t *testing.T) {
	pages := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++
		from, _ := strconv.Atoi(r.URL.Query().Get("from"))
		switch from {
		case 0:
			_, _ = w.Write([]byte(`{"total":25,"hasNext":true,"nextFrom":12,"hits":[{"name":"A","saleUnit":"UN","pricing":{"price":1},"quantity":{"fraction":1}}]}`))
		case 12:
			_, _ = w.Write([]byte(`{"total":25,"hasNext":false,"nextFrom":13,"hits":[{"name":"B","saleUnit":"UN","pricing":{"price":2},"quantity":{"fraction":1}}]}`))
		default:
			t.Fatalf("unexpected from=%d", from)
		}
	}))
	t.Cleanup(srv.Close)
	c := vitrinehttp.Client{
		HTTP: srv.Client(),
		URL: func(q string, from, size int) string {
			return fmt.Sprintf("%s?search=%s&from=%d&size=%d", srv.URL, q, from, size)
		},
		Parse:    fort.ParseSearch,
		PageSize: 12,
	}
	got, err := c.Buscar(context.Background(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if pages != 2 || !got.Esgotada || len(got.Cartoes) != 2 {
		t.Fatalf("pages=%d %#v", pages, got)
	}
}

func TestClient_BuscarCorteDevolveParcial(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		from, _ := strconv.Atoi(r.URL.Query().Get("from"))
		if from == 0 {
			_, _ = w.Write([]byte(`{"total":50,"hasNext":true,"nextFrom":12,"hits":[{"name":"A","saleUnit":"UN","pricing":{"price":1},"quantity":{"fraction":1}}]}`))
			return
		}
		http.Error(w, "down", http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)
	c := vitrinehttp.Client{
		HTTP: srv.Client(),
		URL: func(q string, from, size int) string {
			return fmt.Sprintf("%s?search=%s&from=%d&size=%d", srv.URL, q, from, size)
		},
		Parse: fort.ParseSearch,
	}
	got, err := c.Buscar(context.Background(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if got.Esgotada || len(got.Cartoes) != 1 {
		t.Fatalf("%#v", got)
	}
}
