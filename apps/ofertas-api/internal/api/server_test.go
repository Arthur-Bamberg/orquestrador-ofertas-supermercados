package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-api/internal/config"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store/storetest"
)

func TestMain(m *testing.M) {
	code := m.Run()
	storetest.Stop()
	os.Exit(code)
}

func TestMercadoHandlers(t *testing.T) {
	catalog := storetest.New(t)
	handler := New(catalog, config.Config{CORSOrigin: "http://localhost:5173"})

	req := httptest.NewRequest(http.MethodPost, "/api/mercados", strings.NewReader(`{"id":"m1","nome":"Mercado Um"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("POST status=%d body=%s", res.Code, res.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/mercados/m1", nil)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", res.Code, res.Body.String())
	}
	var got store.Mercado
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "m1" || got.Nome != "Mercado Um" {
		t.Fatalf("got %+v", got)
	}
}

func TestEntityHTTP_CRUDAndConflicts(t *testing.T) {
	catalog := storetest.New(t)
	h := New(catalog, config.Config{CORSOrigin: "http://localhost:5173"})
	if res := doJSON(t, h, http.MethodPost, "/api/mercados", `{"id":"m1","nome":"Mercado Um"}`); res.Code != http.StatusCreated {
		t.Fatalf("mercado POST %d %s", res.Code, res.Body.String())
	}

	t.Run("Fonte_CRUD", func(t *testing.T) {
		res := doJSON(t, h, http.MethodPost, "/api/fontes", `{"id":"f1","mercadoId":"m1","url":"https://example.com"}`)
		if res.Code != http.StatusCreated {
			t.Fatalf("POST status=%d body=%s", res.Code, res.Body.String())
		}
		res = doJSON(t, h, http.MethodGet, "/api/fontes/f1", "")
		if res.Code != http.StatusOK {
			t.Fatalf("GET status=%d body=%s", res.Code, res.Body.String())
		}
		res = doJSON(t, h, http.MethodPut, "/api/fontes/f1", `{"mercadoId":"m1","url":"https://other.example"}`)
		if res.Code != http.StatusOK {
			t.Fatalf("PUT status=%d body=%s", res.Code, res.Body.String())
		}
	})

	t.Run("Produto_Marca_Documento", func(t *testing.T) {
		if res := doJSON(t, h, http.MethodPost, "/api/produtos", `{"id":"p1","nome":"Arroz","nomeNorm":"arroz"}`); res.Code != http.StatusCreated {
			t.Fatalf("produto POST %d %s", res.Code, res.Body.String())
		}
		if res := doJSON(t, h, http.MethodPost, "/api/marcas", `{"id":"ma1","nome":"Camil","nomeNorm":"camil"}`); res.Code != http.StatusCreated {
			t.Fatalf("marca POST %d %s", res.Code, res.Body.String())
		}
		if res := doJSON(t, h, http.MethodPost, "/api/documentos", `{"id":"d1","fonteId":"f1","mercadoId":"m1","filename":"a.pdf","dia":"2026-07-21","estado":"processando"}`); res.Code != http.StatusCreated {
			t.Fatalf("documento POST %d %s", res.Code, res.Body.String())
		}
		if res := doJSON(t, h, http.MethodPost, "/api/fontes", `{"id":"f2","mercadoId":"m1","url":"https://other.example"}`); res.Code != http.StatusCreated {
			t.Fatalf("fonte2 POST %d %s", res.Code, res.Body.String())
		}
		if res := doJSON(t, h, http.MethodPost, "/api/documentos", `{"id":"d2","fonteId":"f2","mercadoId":"m1","filename":"b.pdf","dia":"2026-07-21","estado":"falhou"}`); res.Code != http.StatusCreated {
			t.Fatalf("documento2 POST %d %s", res.Code, res.Body.String())
		}
	})

	t.Run("Documento_filtros", func(t *testing.T) {
		res := doJSON(t, h, http.MethodGet, "/api/documentos?fonteId=f1", "")
		if res.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
		}
		var items []store.Documento
		if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].ID != "d1" {
			t.Fatalf("filtro fonteId=%v", items)
		}
		res = doJSON(t, h, http.MethodGet, "/api/documentos?estado=falhou", "")
		if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].ID != "d2" {
			t.Fatalf("filtro estado=%v", items)
		}
		if res := doJSON(t, h, http.MethodPost, "/api/documentos", `{"id":"d3","fonteId":"f1","mercadoId":"m1","filename":"c.pdf","dia":"2026-08-01","estado":"concluido"}`); res.Code != http.StatusCreated {
			t.Fatalf("documento3 POST %d %s", res.Code, res.Body.String())
		}
		res = doJSON(t, h, http.MethodGet, "/api/documentos?dia=2026-08-01", "")
		if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].ID != "d3" {
			t.Fatalf("filtro dia=%v", items)
		}
	})

	t.Run("Produto_filtros", func(t *testing.T) {
		if res := doJSON(t, h, http.MethodPost, "/api/produtos", `{"id":"p2","nome":"Feijão preto","nomeNorm":"feijao preto","categorias":["mercearia","feijao"]}`); res.Code != http.StatusCreated {
			t.Fatalf("produto2 POST %d %s", res.Code, res.Body.String())
		}
		res := doJSON(t, h, http.MethodGet, "/api/produtos?nome=FEIJ", "")
		if res.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
		}
		var items []store.Produto
		if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].ID != "p2" {
			t.Fatalf("filtro nome=%v", items)
		}
		res = doJSON(t, h, http.MethodGet, "/api/produtos?categoria=feijao", "")
		if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].ID != "p2" {
			t.Fatalf("filtro categoria=%v", items)
		}
	})

	t.Run("Oferta_CRUD_e_filtro", func(t *testing.T) {
		body := `{"id":"o1","produtoId":"p1","mercadoId":"m1","valor":10,"quantidades":[1],"medida":"unidade","dataInicio":"2026-07-21","dataExpiracao":"2026-07-22","documentoIds":["d1"]}`
		res := doJSON(t, h, http.MethodPost, "/api/ofertas", body)
		if res.Code != http.StatusCreated {
			t.Fatalf("POST oferta %d %s", res.Code, res.Body.String())
		}
		res = doJSON(t, h, http.MethodGet, "/api/ofertas?documentoId=d1", "")
		var items []store.Oferta
		if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].ID != "o1" {
			t.Fatalf("filtro documentoId=%v", items)
		}
		res = doJSON(t, h, http.MethodGet, "/api/ofertas?produtoId=p1", "")
		if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 {
			t.Fatalf("filtro produtoId=%v", items)
		}
		body2 := `{"id":"o-feijao","produtoId":"p2","marcaId":"ma1","mercadoId":"m1","valor":8,"quantidades":[1],"medida":"unidade","dataInicio":"2026-07-21","dataExpiracao":"2026-07-22","documentoIds":["d1"]}`
		res = doJSON(t, h, http.MethodPost, "/api/ofertas", body2)
		if res.Code != http.StatusCreated {
			t.Fatalf("POST oferta feijao %d %s", res.Code, res.Body.String())
		}
		res = doJSON(t, h, http.MethodGet, "/api/ofertas?texto=FEIJ", "")
		if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].ID != "o-feijao" {
			t.Fatalf("filtro texto produto=%v", items)
		}
		res = doJSON(t, h, http.MethodGet, "/api/ofertas?texto=camil", "")
		if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].ID != "o-feijao" {
			t.Fatalf("filtro texto marca=%v", items)
		}
	})

	t.Run("Oferta_chave_unica_409", func(t *testing.T) {
		body := `{"id":"o2","produtoId":"p1","mercadoId":"m1","valor":10,"quantidades":[1],"medida":"unidade","dataInicio":"2026-07-21","dataExpiracao":"2026-07-22","documentoIds":["d1"]}`
		res := doJSON(t, h, http.MethodPost, "/api/ofertas", body)
		if res.Code != http.StatusConflict {
			t.Fatalf("status=%d body=%s want 409", res.Code, res.Body.String())
		}
	})

	t.Run("DeleteMercado_bloqueado_409", func(t *testing.T) {
		if res := doJSON(t, h, http.MethodPost, "/api/mercados", `{"id":"m1","nome":"Fort"}`); res.Code != http.StatusCreated {
			t.Fatalf("mercado POST %d %s", res.Code, res.Body.String())
		}
		res := doJSON(t, h, http.MethodDelete, "/api/mercados/m1", "")
		if res.Code != http.StatusConflict {
			t.Fatalf("DELETE mercado status=%d body=%s want 409", res.Code, res.Body.String())
		}
	})

	t.Run("UsoExtrator_CRUD", func(t *testing.T) {
		res := doJSON(t, h, http.MethodPost, "/api/usos-extrator", `{"documentoId":"d1","tentativa":"t1","artefatoPath":"p","model":"stub","promptTokens":3}`)
		if res.Code != http.StatusCreated {
			t.Fatalf("POST uso %d %s", res.Code, res.Body.String())
		}
		res = doJSON(t, h, http.MethodGet, "/api/usos-extrator/d1/t1", "")
		if res.Code != http.StatusOK {
			t.Fatalf("GET uso %d %s", res.Code, res.Body.String())
		}
	})
}

func TestCORSOptions(t *testing.T) {
	catalog := storetest.New(t)
	h := New(catalog, config.Config{CORSOrigin: "http://localhost:5173"})
	res := doJSON(t, h, http.MethodOptions, "/api/mercados", "")
	if res.Code != http.StatusNoContent {
		t.Fatalf("status=%d", res.Code)
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("CORS origin=%q", got)
	}
}

func TestOperacaoPipelineHTTP(t *testing.T) {
	catalog := storetest.New(t)
	h := New(catalog, config.Config{})

	res := doJSON(t, h, http.MethodPost, "/api/ops/run", "{}")
	if res.Code != http.StatusAccepted {
		t.Fatalf("enqueue run %d %s", res.Code, res.Body.String())
	}
	var run store.OperacaoPipeline
	if err := json.NewDecoder(res.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	if run.Kind != store.OperacaoRun || run.Status != store.OperacaoPending {
		t.Fatalf("run=%+v", run)
	}

	res = doJSON(t, h, http.MethodPost, "/api/ops/discover", `{"fonteId":"f1"}`)
	if res.Code != http.StatusAccepted {
		t.Fatalf("discover %d %s", res.Code, res.Body.String())
	}

	res = doJSON(t, h, http.MethodGet, "/api/ops", "")
	if res.Code != http.StatusOK {
		t.Fatalf("list ops %d %s", res.Code, res.Body.String())
	}
	var ops []store.OperacaoPipeline
	if err := json.NewDecoder(res.Body).Decode(&ops); err != nil {
		t.Fatal(err)
	}
	if len(ops) < 2 {
		t.Fatalf("ops=%v", ops)
	}

	res = doJSON(t, h, http.MethodPost, "/api/ops/"+run.ID+"/cancel", "")
	if res.Code != http.StatusOK {
		t.Fatalf("cancel %d %s", res.Code, res.Body.String())
	}
	var canceled store.OperacaoPipeline
	if err := json.NewDecoder(res.Body).Decode(&canceled); err != nil {
		t.Fatal(err)
	}
	if canceled.Status != store.OperacaoCanceled {
		t.Fatalf("status=%s", canceled.Status)
	}
}

func TestArtefatosHTTP(t *testing.T) {
	catalog := storetest.New(t)
	root := t.TempDir()
	h := New(catalog, config.Config{ArtefatoRoot: root})
	if res := doJSON(t, h, http.MethodPost, "/api/mercados", `{"id":"m1","nome":"M"}`); res.Code != http.StatusCreated {
		t.Fatalf("mercado %d %s", res.Code, res.Body.String())
	}
	if res := doJSON(t, h, http.MethodPost, "/api/fontes", `{"id":"f1","mercadoId":"m1","url":"https://example.com"}`); res.Code != http.StatusCreated {
		t.Fatalf("fonte %d %s", res.Code, res.Body.String())
	}
	if res := doJSON(t, h, http.MethodPost, "/api/documentos", `{"id":"d1","fonteId":"f1","mercadoId":"m1","filename":"a.pdf","dia":"2026-07-21"}`); res.Code != http.StatusCreated {
		t.Fatalf("documento %d %s", res.Code, res.Body.String())
	}

	res := doJSON(t, h, http.MethodGet, "/api/documentos/d1/artefatos", "")
	if res.Code != http.StatusOK {
		t.Fatalf("list empty %d %s", res.Code, res.Body.String())
	}

	put := httptest.NewRequest(http.MethodPut, "/api/documentos/d1/artefatos/t1/raw.json", strings.NewReader(`{"ok":true}`))
	putRes := httptest.NewRecorder()
	h.ServeHTTP(putRes, put)
	if putRes.Code != http.StatusOK {
		t.Fatalf("upload %d %s", putRes.Code, putRes.Body.String())
	}

	res = doJSON(t, h, http.MethodGet, "/api/documentos/d1/artefatos", "")
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "raw.json") {
		t.Fatalf("list after upload %d %s", res.Code, res.Body.String())
	}

	get := httptest.NewRequest(http.MethodGet, "/api/documentos/d1/artefatos/t1/raw.json", nil)
	getRes := httptest.NewRecorder()
	h.ServeHTTP(getRes, get)
	if getRes.Code != http.StatusOK || !strings.Contains(getRes.Body.String(), `"ok":true`) {
		t.Fatalf("download %d %s", getRes.Code, getRes.Body.String())
	}

	del := doJSON(t, h, http.MethodDelete, "/api/documentos/d1/artefatos/t1/raw.json", "")
	if del.Code != http.StatusNoContent {
		t.Fatalf("delete file %d %s", del.Code, del.Body.String())
	}
}

func doJSON(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}
