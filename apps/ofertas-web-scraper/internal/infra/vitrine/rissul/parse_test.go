package rissul_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine/rissul"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestSearchURL_VTEXCatalog(t *testing.T) {
	got := rissul.SearchURL("arroz integral")
	if !strings.Contains(got, "www.rissul.com.br/api/catalog_system/pub/products/search") {
		t.Fatalf("got %s", got)
	}
	if !strings.Contains(got, "ft=arroz") {
		t.Fatalf("got %s", got)
	}
}

func TestParseSearch_VTEXPriceAndMarca(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "testdata", "search.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := rissul.ParseSearch(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d", len(got))
	}
	if got[0].Nome != "Arroz Integral Vapza 200g" || got[0].Marca != "VAPZA" {
		t.Fatalf("%#v", got[0])
	}
	if got[0].Valor != 9.79 || got[0].IndicacaoPromocional {
		t.Fatalf("price/promo %#v", got[0])
	}
	if got[0].Medida != store.MedidaUnidade || got[0].Quantidades[0] != 1 {
		t.Fatalf("medida %#v", got[0])
	}
	if got[1].Nome != "Arroz Integral Ynagi Cateto Japonês 1kg" || got[1].Valor != 12.95 || got[1].Marca != "YANAGI" {
		t.Fatalf("%#v", got[1])
	}
}

func TestParseSearch_ListPriceMaiorQuePriceEIndicacao(t *testing.T) {
	body := []byte(`[
	  {"productName":"Arroz Integral Camil 1kg","brand":"Camil","items":[{"measurementUnit":"un","unitMultiplier":1,
	    "sellers":[{"commertialOffer":{"Price":8.9,"ListPrice":10.9}}]}]}
	]`)
	got, err := rissul.ParseSearch(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Valor != 8.9 || !got[0].IndicacaoPromocional {
		t.Fatalf("%#v", got)
	}
}
