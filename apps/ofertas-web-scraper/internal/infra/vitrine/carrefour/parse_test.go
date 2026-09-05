package carrefour_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine/carrefour"
)

func TestParseSearch_VTEXPriceAndPromo(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "testdata", "search.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := carrefour.ParseSearch(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Valor != 9.49 || !got[0].IndicacaoPromocional || got[0].Marca != "Camil" {
		t.Fatalf("%#v", got)
	}
}
