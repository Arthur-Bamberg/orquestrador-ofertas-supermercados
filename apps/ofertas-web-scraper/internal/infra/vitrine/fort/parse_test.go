package fort_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine/fort"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestParseSearch_LePrecoMarcaEIndicacao(t *testing.T) {
	body := readFixture(t, "search.json")
	got, err := fort.ParseSearch(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d", len(got))
	}
	if got[0].Nome != "Arroz Integral Camil 1kg" || got[0].Marca != "Camil" || got[0].Valor != 8.9 {
		t.Fatalf("%#v", got[0])
	}
	if got[0].Medida != store.MedidaG || got[0].Quantidades[0] != 1000 {
		t.Fatalf("medida %#v", got[0])
	}
	if !got[0].IndicacaoPromocional {
		t.Fatal("listPrice > price is promotional")
	}
	if got[1].IndicacaoPromocional {
		t.Fatal("equal prices are not promotional")
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
