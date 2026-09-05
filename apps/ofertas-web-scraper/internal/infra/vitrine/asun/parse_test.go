package asun_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine/asun"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestSearchURL_CavalhadaSense(t *testing.T) {
	got := asun.SearchURL("arroz integral")
	if !strings.HasPrefix(got, "https://sense.osuper.com.br/319/1540/search?") {
		t.Fatalf("got %s", got)
	}
}

func TestParseSearch_LePrecoMarcaEIndicacao(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "testdata", "search.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := asun.ParseSearch(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d", len(got))
	}
	if got[0].Nome != "Leite Integral Zero Lactose Tirol 1L" || got[0].Marca != "Tirol" {
		t.Fatalf("%#v", got[0])
	}
	if got[0].Valor != 4.89 || !got[0].IndicacaoPromocional {
		t.Fatalf("promo card %#v", got[0])
	}
	if got[0].Medida != store.MedidaUnidade {
		t.Fatalf("medida %#v", got[0])
	}
	if got[1].Nome != "Leite longa vida Dália integral 1l" || got[1].Valor != 4.95 || got[1].IndicacaoPromocional {
		t.Fatalf("regular card %#v", got[1])
	}
}
