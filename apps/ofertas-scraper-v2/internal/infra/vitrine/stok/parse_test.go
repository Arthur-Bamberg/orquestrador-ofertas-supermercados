package stok_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrine/stok"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestSearchURL_LojaPadraoDoSite(t *testing.T) {
	got := stok.SearchURL("tomate", 0, 100)
	if !strings.HasPrefix(got, "https://services.vipcommerce.com.br/api-admin/v1/org/130/filial/1/centro_distribuicao/1/loja/buscas/produtos/termo/tomate?") {
		t.Fatalf("got %s", got)
	}
	if !strings.Contains(got, "page=1") {
		t.Fatalf("got %s", got)
	}
}

func TestParseSearch_LePrecoMarcaEIndicacao(t *testing.T) {
	pagina, err := stok.ParseSearch(readFixture(t, "search.json"))
	if err != nil {
		t.Fatal(err)
	}
	if pagina.Total != 80 {
		t.Fatalf("total=%d", pagina.Total)
	}
	if pagina.HasNext {
		t.Fatal("single page must not HasNext")
	}
	got := pagina.Cartoes
	if len(got) != 3 {
		t.Fatalf("got %d", len(got))
	}
	if got[0].Nome != "Tomate Longa Vida Kg (aprox. 800g)" || got[0].Marca != "" {
		t.Fatalf("%#v", got[0])
	}
	if got[0].Valor != 11.39 {
		t.Fatalf("KG shelf is per-kg 11.39, got %v", got[0].Valor)
	}
	if got[0].IndicacaoPromocional {
		t.Fatal("preco_original on KG is /kg display, not IndicacaoPromocional")
	}
	if got[0].Medida != store.MedidaG || got[0].Quantidades[0] != 1000 {
		t.Fatalf("medida %#v", got[0])
	}
	if got[1].Nome != "Tomate Sweet Grape 180g" || got[1].Valor != 10.19 || got[1].IndicacaoPromocional {
		t.Fatalf("regular card %#v", got[1])
	}
	if got[1].Medida != store.MedidaUnidade || got[1].Quantidades[0] != 1 {
		t.Fatalf("UN %#v", got[1])
	}
	if got[2].Nome != "Passata Corper Rústica Tomate Triturado 1020g" || got[2].Valor != 16.99 {
		t.Fatalf("clube shelf price want 16.99, got %#v", got[2])
	}
	if !got[2].IndicacaoPromocional {
		t.Fatal("em_oferta is IndicacaoPromocional")
	}
}

func TestTokenProvider_HeaderComJWTDaLoja(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.Header.Get(stok.OrganizationIDHeader) != stok.OrgID {
			t.Fatalf("OrganizationId=%q", r.Header.Get(stok.OrganizationIDHeader))
		}
		if r.Header.Get(stok.DomainKeyHeader) != stok.DomainKey {
			t.Fatalf("DomainKey=%q", r.Header.Get(stok.DomainKeyHeader))
		}
		var body struct {
			Domain   string `json:"domain"`
			Username string `json:"username"`
			Key      string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Domain != stok.DomainKey || body.Username != "loja" || body.Key == "" {
			t.Fatalf("login %#v", body)
		}
		_, _ = w.Write([]byte(`{"success":true,"data":"jwt-stok-test"}`))
	}))
	t.Cleanup(srv.Close)
	p := &stok.TokenProvider{HTTP: srv.Client(), LoginURL: srv.URL}
	h, err := p.Header(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := h.Get("Authorization"); got != "Bearer jwt-stok-test" {
		t.Fatalf("Authorization=%q", got)
	}
	if h.Get(stok.OrganizationIDHeader) != stok.OrgID || h.Get(stok.DomainKeyHeader) != stok.DomainKey {
		t.Fatalf("headers=%v", h)
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
