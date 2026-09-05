package rissul_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine/rissul"
)

func TestParseSearch_PrecoDeIndicaPromocao(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "testdata", "search.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := rissul.ParseSearch(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !got[0].IndicacaoPromocional || got[1].IndicacaoPromocional {
		t.Fatalf("%#v", got)
	}
	if got[0].Valor != 8.9 || got[0].Marca != "Camil" {
		t.Fatalf("%#v", got[0])
	}
}
