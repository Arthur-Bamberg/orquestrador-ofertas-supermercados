package fort_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrine/fort"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestSearchURL_LojaSenseDoSitePadrao(t *testing.T) {
	got := fort.SearchURL("tomate", 0, 12)
	if strings.Contains(got, "1687") {
		t.Fatalf("Canoas 1687 is not the site default: %s", got)
	}
	if !strings.HasPrefix(got, "https://sense.osuper.com.br/334/1585/search?") {
		t.Fatalf("got %s", got)
	}
	if !strings.Contains(got, "search=tomate") || !strings.Contains(got, "size=12") || !strings.Contains(got, "from=0") {
		t.Fatalf("got %s", got)
	}
}

func TestParseSearch_LePrecoMarcaEIndicacao(t *testing.T) {
	body := readFixture(t, "search.json")
	pagina, err := fort.ParseSearch(body)
	if err != nil {
		t.Fatal(err)
	}
	got := pagina.Cartoes
	if pagina.Total != 82 {
		t.Fatalf("total=%d", pagina.Total)
	}
	if len(got) != 2 {
		t.Fatalf("got %d", len(got))
	}
	if got[0].Nome != "Leite UHT Tirol Semi Desnatado com Tampa Rosca 1L" || got[0].Marca != "Tirol" {
		t.Fatalf("%#v", got[0])
	}
	if got[0].Valor != 3.89 {
		t.Fatalf("promotional shelf price want 3.89, got %v", got[0].Valor)
	}
	if !got[0].IndicacaoPromocional {
		t.Fatal("pricing.promotion is IndicacaoPromocional")
	}
	if got[0].Medida != store.MedidaUnidade || got[0].Quantidades[0] != 1 {
		t.Fatalf("medida %#v", got[0])
	}
	if got[1].Nome != "LEITE SEMIDESN DALIA 1L UHT ZERO LACTOSE" || got[1].Marca != "Dalia" {
		t.Fatalf("%#v", got[1])
	}
	if got[1].Valor != 5.48 || got[1].IndicacaoPromocional {
		t.Fatalf("regular card %#v", got[1])
	}
}

func TestTokenProvider_LeSearchEngineTokenDoHome(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"searchEngineToken":"v1.334.1585.test-token","x":1}`))
	}))
	t.Cleanup(srv.Close)
	p := &fort.TokenProvider{HTTP: srv.Client(), HomeURL: srv.URL}
	h, err := p.Header(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := h.Get(fort.SearchTokenHeader); got != "v1.334.1585.test-token" {
		t.Fatalf("header=%q", got)
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
