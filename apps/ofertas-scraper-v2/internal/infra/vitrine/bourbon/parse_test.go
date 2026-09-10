package bourbon_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrine/bourbon"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestSearchURL_LojaPadraoDoSite(t *testing.T) {
	got := bourbon.SearchURL("tomate", 0, 12)
	if strings.Contains(strings.ToLower(got), "canoas") {
		t.Fatalf("Canoas is not the site default: %s", got)
	}
	if !strings.HasPrefix(got, "https://www.zaffari.com.br/api/io/_v/api/intelligent-search/product_search/?") {
		t.Fatalf("got %s", got)
	}
	if !strings.Contains(got, "query=tomate") || !strings.Contains(got, "count=12") || !strings.Contains(got, "page=1") {
		t.Fatalf("got %s", got)
	}
	if !strings.Contains(got, "hideUnavailableItems=true") {
		t.Fatalf("vitrine hides unavailable cards: %s", got)
	}
}

func TestParseSearch_LePrecoMarcaEIndicacao(t *testing.T) {
	pagina, err := bourbon.ParseSearch(readFixture(t, "search.json"))
	if err != nil {
		t.Fatal(err)
	}
	if pagina.Total != 24 {
		t.Fatalf("total=%d", pagina.Total)
	}
	if !pagina.HasNext {
		t.Fatal("page 1 of 2 must HasNext")
	}
	got := pagina.Cartoes
	if len(got) != 3 {
		t.Fatalf("got %d", len(got))
	}
	if got[0].Nome != "Tomate Italiano" || got[0].Marca != "Hortifrutigranjeiros" {
		t.Fatalf("%#v", got[0])
	}
	if got[0].Valor != 14.9 {
		t.Fatalf("KG shelf is per-kg 14.9, got %v", got[0].Valor)
	}
	if got[0].IndicacaoPromocional {
		t.Fatal("equal list and selling price is not IndicacaoPromocional")
	}
	if got[0].Medida != store.MedidaG || got[0].Quantidades[0] != 1000 {
		t.Fatalf("medida %#v", got[0])
	}
	if got[1].Nome != "Tomate Grape Doce 180g" || got[1].Valor != 6.49 || got[1].IndicacaoPromocional {
		t.Fatalf("regular card %#v", got[1])
	}
	if got[1].Medida != store.MedidaUnidade || got[1].Quantidades[0] != 1 {
		t.Fatalf("UN %#v", got[1])
	}
	if got[2].Nome != "Tomate Gaúcho 700g" || got[2].Valor != 16.9 {
		t.Fatalf("regular card %#v", got[2])
	}
}

func TestParseSearch_ListPriceMaiorQueAtualEIndicacao(t *testing.T) {
	pagina, err := bourbon.ParseSearch([]byte(`{
		"products":[{"productName":"Cenoura","brand":"Hortifrutigranjeiros","items":[{
			"measurementUnit":"kg","unitMultiplier":0.2,
			"sellers":[{"commertialOffer":{"Price":5.49,"ListPrice":7.98}}]
		}]}],
		"recordsFiltered":1,
		"pagination":{"current":{"index":1},"next":{"index":0},"perPage":12}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if pagina.HasNext {
		t.Fatal("single page must not HasNext")
	}
	if len(pagina.Cartoes) != 1 {
		t.Fatalf("got %d", len(pagina.Cartoes))
	}
	got := pagina.Cartoes[0]
	if got.Valor != 5.49 {
		t.Fatalf("selling Price want 5.49, got %v", got.Valor)
	}
	if !got.IndicacaoPromocional {
		t.Fatal("ListPrice > Price is IndicacaoPromocional")
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	b, err := os.ReadFile(filepath.Join(filepath.Dir(file), "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}
