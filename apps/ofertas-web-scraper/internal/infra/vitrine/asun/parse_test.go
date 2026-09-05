package asun_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine/asun"
)

func TestParseSearch_HTMLCardsAndPromoFlag(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "testdata", "search.html"))
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
	if got[0].Valor != 8.9 || !got[0].IndicacaoPromocional || got[0].Marca != "Camil" {
		t.Fatalf("%#v", got[0])
	}
	if got[1].IndicacaoPromocional {
		t.Fatal("second card is not promotional")
	}
}
