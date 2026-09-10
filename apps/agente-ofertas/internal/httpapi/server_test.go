package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/httpapi"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestHTTP_healthListasInterpretarEMCP(t *testing.T) {
	envio := &stubEnvio{}
	ag := application.New(application.Deps{Cat: catLeite(), Coleta: coletaNop{}, Envio: envio, Hoje: hoje})
	h := httpapi.New(ag, "secret")

	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/health", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("health %d", res.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/listas", bytes.NewBufferString(`{"conversaJid":"5511999999999","corpo":"leite"}`))
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("sem token %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/listas", bytes.NewBufferString(`{"conversaJid":"5511999999999","corpo":"leite"}`))
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusAccepted {
		t.Fatalf("listas %d %s", res.Code, res.Body.String())
	}
	if len(envio.calls) != 1 || envio.calls[0] != "5511999999999" {
		t.Fatalf("%+v", envio.calls)
	}

	req = httptest.NewRequest(http.MethodPost, "/interpretar", bytes.NewBufferString(`{"texto":"xyzabc"}`))
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("interpretar %d", res.Code)
	}
	var out map[string]string
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out["resposta"] != "*xyzabc*\nNão achei." {
		t.Fatalf("%q", out["resposta"])
	}

	req = httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !bytes.Contains(res.Body.Bytes(), []byte("interpretar_lista")) {
		t.Fatalf("mcp list %d %s", res.Code, res.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"interpretar_lista","arguments":{"texto":"xyzabc"}}}`))
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !bytes.Contains(res.Body.Bytes(), []byte("Não achei")) {
		t.Fatalf("mcp call %d %s", res.Code, res.Body.String())
	}
}

func hoje() time.Time { return time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC) }

type stubEnvio struct{ calls []string }

func (s *stubEnvio) Enviar(_ context.Context, conversaJID, _ string) error {
	s.calls = append(s.calls, conversaJID)
	return nil
}

type memCat struct {
	produtos []store.Produto
	marcas   []store.Marca
	ofertas  []store.Oferta
	mercados map[store.MercadoID]store.Mercado
}

func (m memCat) ListarProdutos(context.Context) ([]store.Produto, error) { return m.produtos, nil }
func (m memCat) ListarMarcas(context.Context) ([]store.Marca, error)     { return m.marcas, nil }
func (m memCat) ListarOfertas(context.Context) ([]store.Oferta, error)   { return m.ofertas, nil }
func (m memCat) GetMercado(_ context.Context, id store.MercadoID) (store.Mercado, bool, error) {
	x, ok := m.mercados[id]
	return x, ok, nil
}

type coletaNop struct{}

func (coletaNop) Coletar(context.Context, string) ([]store.Oferta, error) { return nil, nil }

func catLeite() memCat {
	return memCat{
		produtos: []store.Produto{{ID: "leite", Nome: "Leite integral", NomeNorm: "leite integral"}},
		ofertas: []store.Oferta{{
			ID: "o1", ProdutoID: "leite", MercadoID: "fort", DocumentoID: "d1",
			Valor: 5.9, Quantidades: []float64{1000}, Medida: store.MedidaML,
			DataInicio: "2026-09-01", DataExpiracao: "2026-09-10",
		}},
		mercados: map[store.MercadoID]store.Mercado{"fort": {ID: "fort", Nome: "Fort"}},
	}
}
