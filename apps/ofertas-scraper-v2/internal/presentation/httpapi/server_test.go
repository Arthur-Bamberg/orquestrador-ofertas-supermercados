package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/presentation/httpapi"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store/storetest"
)

func TestMain(m *testing.M) {
	code := m.Run()
	storetest.Stop()
	os.Exit(code)
}

type fakeVitrine struct {
	cartoes []domain.CartaoVitrine
	calls   *int
}

func (f fakeVitrine) Buscar(context.Context, string) (domain.ResultadoVitrine, error) {
	if f.calls != nil {
		*f.calls++
	}
	return domain.ResultadoVitrine{Cartoes: f.cartoes, Esgotada: true}, nil
}

func TestPOSTColetas_DevolveOfertasPersistidas(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	if err := catalog.SaveMercado(ctx, store.Mercado{ID: "mercado-fort", Nome: "Fort Atacadista"}); err != nil {
		t.Fatal(err)
	}
	h := httpapi.New(httpapi.Deps{
		Catalog: catalog,
		Vitrines: map[store.MercadoID]domain.Vitrine{
			"mercado-fort": fakeVitrine{cartoes: []domain.CartaoVitrine{
				{Nome: "Tomate 1kg", Valor: 6.9, Quantidades: []float64{1000}, Medida: store.MedidaG, IndicacaoPromocional: true},
			}},
		},
		Hoje:  func() string { return "2026-09-07" },
		Token: "secret",
	})

	req := httptest.NewRequest(http.MethodPost, "/coletas", bytes.NewBufferString(`{"termo":"tomate"}`))
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("sem token %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/coletas", bytes.NewBufferString(`{"termo":"tomate"}`))
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("coletas %d %s", res.Code, res.Body.String())
	}
	var out struct {
		Ofertas []store.Oferta `json:"ofertas"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Ofertas) != 1 || out.Ofertas[0].Valor != 6.9 {
		t.Fatalf("got %#v", out.Ofertas)
	}
	coleta, ok, err := catalog.GetColetaByIdentity(ctx, "tomate", "mercado-fort", "2026-09-07")
	if err != nil || !ok {
		t.Fatalf("persisted coleta ok=%v err=%v", ok, err)
	}
	list, err := store.NewOfertaRepo(catalog).ListByColeta(ctx, coleta.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Valor != 6.9 {
		t.Fatalf("catalog %#v", list)
	}
}

func TestPOSTColetas_MercadoIdRestringeVitrine(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	if err := catalog.SaveMercado(ctx, store.Mercado{ID: "mercado-fort", Nome: "Fort Atacadista"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveMercado(ctx, store.Mercado{ID: "mercado-via", Nome: "Via Atacadista"}); err != nil {
		t.Fatal(err)
	}
	viaCalls := 0
	h := httpapi.New(httpapi.Deps{
		Catalog: catalog,
		Vitrines: map[store.MercadoID]domain.Vitrine{
			"mercado-fort": fakeVitrine{cartoes: []domain.CartaoVitrine{
				{Nome: "Tomate 1kg", Valor: 6.9, Quantidades: []float64{1000}, Medida: store.MedidaG},
			}},
			"mercado-via": fakeVitrine{calls: &viaCalls, cartoes: []domain.CartaoVitrine{
				{Nome: "Tomate Via", Valor: 4.5, Quantidades: []float64{1000}, Medida: store.MedidaG},
			}},
		},
		Hoje:  func() string { return "2026-09-07" },
		Token: "secret",
	})

	req := httptest.NewRequest(http.MethodPost, "/coletas", bytes.NewBufferString(`{"termo":"tomate","mercadoId":"mercado-fort"}`))
	req.Header.Set("Authorization", "Bearer secret")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("coletas %d %s", res.Code, res.Body.String())
	}
	var out struct {
		Ofertas []store.Oferta `json:"ofertas"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Ofertas) != 1 || out.Ofertas[0].MercadoID != "mercado-fort" || out.Ofertas[0].Valor != 6.9 {
		t.Fatalf("got %#v", out.Ofertas)
	}
	if viaCalls != 0 {
		t.Fatalf("Via vitrine must not be searched, calls=%d", viaCalls)
	}

	req = httptest.NewRequest(http.MethodPost, "/coletas", bytes.NewBufferString(`{"termo":"tomate","mercadoId":"mercado-stok"}`))
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("sem vitrine %d %s", res.Code, res.Body.String())
	}
}

func TestGETHealth(t *testing.T) {
	h := httpapi.New(httpapi.Deps{})
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/health", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("health %d", res.Code)
	}
}
