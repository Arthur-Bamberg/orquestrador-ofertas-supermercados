package carrefour_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine/carrefour"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestSearchURL_VTEXCatalog(t *testing.T) {
	got := carrefour.SearchURL("arroz integral")
	if !strings.Contains(got, "www.carrefour.com.br/api/catalog_system/pub/products/search/") {
		t.Fatalf("got %s", got)
	}
	if !strings.Contains(got, "/search/arroz") {
		t.Fatalf("got %s", got)
	}
}

func TestParseSearch_VTEXPriceAndMarca(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "testdata", "search.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := carrefour.ParseSearch(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d %#v", len(got), got)
	}
	if got[0].Nome != "Kit 2 Biscoito Arroz Integral Camil Com Pimenta 150g" {
		t.Fatalf("%#v", got[0])
	}
	if got[0].Valor != 29.18 || got[0].IndicacaoPromocional {
		t.Fatalf("price/promo %#v", got[0])
	}
	if got[0].Marca != "" {
		t.Fatalf("Não Informado is not a Marca, got %q", got[0].Marca)
	}
	if got[0].Medida != store.MedidaUnidade || got[0].Quantidades[0] != 1 {
		t.Fatalf("medida %#v", got[0])
	}
	if got[1].Nome != "Bolacha De Arroz Integral Name 130g" || got[1].Valor != 34.22 {
		t.Fatalf("%#v", got[1])
	}
}

func TestParseSearch_ListPriceMaiorQuePriceEIndicacao(t *testing.T) {
	body := []byte(`[
	  {"productName":"Arroz Integral Camil 1kg","brand":"Camil","items":[{"measurementUnit":"un","unitMultiplier":1,
	    "sellers":[{"commertialOffer":{"Price":9.49,"ListPrice":11.99}}]}]}
	]`)
	got, err := carrefour.ParseSearch(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Valor != 9.49 || !got[0].IndicacaoPromocional || got[0].Marca != "Camil" {
		t.Fatalf("%#v", got)
	}
}
