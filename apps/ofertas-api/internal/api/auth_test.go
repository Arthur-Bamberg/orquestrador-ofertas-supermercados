package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-api/internal/config"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store/storetest"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/operador"
)

const senhaTeste = "senha-teste"

func novaAPI(t *testing.T, catalog *store.Catalog, cfg config.Config) http.Handler {
	t.Helper()
	ids := abrirIDs(t)
	if _, err := ids.Criar(context.Background(), "arthur", senhaTeste); err != nil {
		t.Fatal(err)
	}
	h := New(catalog, ids, cfg)
	cookie := cookieIdentificar(t, h, "arthur", senhaTeste)
	return comCookie(h, cookie)
}

func abrirIDs(t *testing.T) *operador.PG {
	t.Helper()
	_ = storetest.DSN(t)
	ctx := context.Background()
	ids, err := operador.Open(ctx, storetest.DSN(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ids.Close)
	if err := ids.Truncate(ctx); err != nil {
		t.Fatal(err)
	}
	return ids
}

func cookieIdentificar(t *testing.T, h http.Handler, nome, senha string) *http.Cookie {
	t.Helper()
	body := `{"nome":"` + nome + `","senha":"` + senha + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/identificar", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("identificar %d %s", res.Code, res.Body.String())
	}
	for _, c := range res.Result().Cookies() {
		if c.Name == operador.CookieName {
			if !c.HttpOnly || c.Path != "/" {
				t.Fatalf("cookie %+v", c)
			}
			return c
		}
	}
	t.Fatal("cookie do Operador ausente")
	return nil
}

func comCookie(h http.Handler, c *http.Cookie) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.AddCookie(c)
		h.ServeHTTP(w, r)
	})
}

func TestIdentificar_catalogoExigeOperador(t *testing.T) {
	catalog := storetest.New(t)
	ids := abrirIDs(t)
	if _, err := ids.Criar(context.Background(), "arthur", senhaTeste); err != nil {
		t.Fatal(err)
	}
	h := New(catalog, ids, config.Config{})

	res := doJSON(t, h, http.MethodGet, "/api/mercados", "")
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("anônimo %d", res.Code)
	}

	res = doJSON(t, h, http.MethodPost, "/api/identificar", `{"nome":"arthur","senha":"errada"}`)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("senha %d %s", res.Code, res.Body.String())
	}

	cookie := cookieIdentificar(t, h, "arthur", senhaTeste)
	req := httptest.NewRequest(http.MethodGet, "/api/mercados", nil)
	req.AddCookie(cookie)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("identificado %d %s", res.Code, res.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/eu", nil)
	req.AddCookie(cookie)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"arthur"`) {
		t.Fatalf("eu %d %s", res.Code, res.Body.String())
	}
}

func TestRequireOperador_todasAsRotasDoBackoffice(t *testing.T) {
	catalog := storetest.New(t)
	ids := abrirIDs(t)
	h := New(catalog, ids, config.Config{})
	rotas := []struct{ method, path string }{
		{http.MethodGet, "/api/eu"},
		{http.MethodGet, "/api/mercados"},
		{http.MethodPost, "/api/mercados"},
		{http.MethodGet, "/api/mercados/x"},
		{http.MethodPut, "/api/mercados/x"},
		{http.MethodDelete, "/api/mercados/x"},
		{http.MethodGet, "/api/fontes"},
		{http.MethodPost, "/api/fontes"},
		{http.MethodGet, "/api/produtos"},
		{http.MethodGet, "/api/marcas"},
		{http.MethodGet, "/api/documentos"},
		{http.MethodGet, "/api/documentos/x/falhas"},
		{http.MethodGet, "/api/documentos/x/artefatos"},
		{http.MethodGet, "/api/ofertas"},
		{http.MethodPost, "/api/ofertas"},
		{http.MethodGet, "/api/usos-extrator"},
		{http.MethodGet, "/api/ops"},
		{http.MethodPost, "/api/ops/run"},
		{http.MethodPost, "/api/ops/discover"},
		{http.MethodPost, "/api/ops/reprocess"},
		{http.MethodPost, "/api/ops/testar"},
		{http.MethodGet, "/api/ops/x"},
		{http.MethodPost, "/api/ops/x/cancel"},
	}
	for _, r := range rotas {
		res := doJSON(t, h, r.method, r.path, "{}")
		if res.Code != http.StatusUnauthorized {
			t.Errorf("%s %s anônimo %d %s", r.method, r.path, res.Code, res.Body.String())
		}
	}
}

func TestSair_naoDerrubaOutraIdentificacao(t *testing.T) {
	catalog := storetest.New(t)
	ids := abrirIDs(t)
	if _, err := ids.Criar(context.Background(), "arthur", senhaTeste); err != nil {
		t.Fatal(err)
	}
	h := New(catalog, ids, config.Config{})
	a := cookieIdentificar(t, h, "arthur", senhaTeste)
	b := cookieIdentificar(t, h, "arthur", senhaTeste)

	req := httptest.NewRequest(http.MethodPost, "/api/sair", nil)
	req.AddCookie(a)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("sair %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/eu", nil)
	req.AddCookie(a)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("A após sair %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/eu", nil)
	req.AddCookie(b)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("B %d %s", res.Code, res.Body.String())
	}
}

func TestApagarOperador_derrubaIdentificacoes(t *testing.T) {
	catalog := storetest.New(t)
	ids := abrirIDs(t)
	op, err := ids.Criar(context.Background(), "maria", senhaTeste)
	if err != nil {
		t.Fatal(err)
	}
	h := New(catalog, ids, config.Config{})
	a := cookieIdentificar(t, h, "maria", senhaTeste)
	b := cookieIdentificar(t, h, "maria", senhaTeste)
	if err := ids.Apagar(context.Background(), op.ID); err != nil {
		t.Fatal(err)
	}
	for _, c := range []*http.Cookie{a, b} {
		req := httptest.NewRequest(http.MethodGet, "/api/eu", nil)
		req.AddCookie(c)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("ainda identificado %d", res.Code)
		}
	}
}

func TestTrocarSenha_naoDerrubaIdentificacao(t *testing.T) {
	catalog := storetest.New(t)
	ids := abrirIDs(t)
	op, err := ids.Criar(context.Background(), "arthur", senhaTeste)
	if err != nil {
		t.Fatal(err)
	}
	h := New(catalog, ids, config.Config{})
	cookie := cookieIdentificar(t, h, "arthur", senhaTeste)
	if err := ids.DefinirSenha(context.Background(), op.ID, "outra"); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/eu", nil)
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("após trocar senha %d", res.Code)
	}
	res = doJSON(t, h, http.MethodPost, "/api/identificar", `{"nome":"arthur","senha":"`+senhaTeste+`"}`)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("senha antiga %d", res.Code)
	}
}
