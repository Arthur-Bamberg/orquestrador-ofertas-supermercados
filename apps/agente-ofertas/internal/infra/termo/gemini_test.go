package termo_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/infra/termo"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestGemini_TermoDevolveTokensDeBusca(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{{
				"content": map[string]any{
					"parts": []map[string]string{{"text": `{"termo":"cenoura orgânica"}`}},
				},
			}},
		})
	}))
	t.Cleanup(srv.Close)

	g := termo.Gemini{Key: "k", Model: "gemini-3-flash-preview", Base: srv.URL, HTTP: srv.Client()}
	got, err := g.Termo(context.Background(), "Quero 2kg de cenoura orgânica")
	if err != nil {
		t.Fatal(err)
	}
	if got != "cenoura orgânica" {
		t.Fatalf("%q", got)
	}
	if !strings.Contains(string(gotBody), "Quero 2kg de cenoura orgânica") {
		t.Fatalf("pedido sem o Item: %s", gotBody)
	}
	pedido := string(gotBody)
	if !strings.Contains(pedido, "tamanho") || strings.Contains(pedido, "Não inclua verbo de compra, cumprimento, quantidade, unidade de medida") {
		t.Fatalf("instrução descarta tamanho: %s", pedido)
	}
	if !strings.Contains(pedido, "creme de leite") {
		t.Fatalf("instrução tira preposição do tipo: %s", pedido)
	}
}

func TestGemini_ClassificarConsultaDevolveTexto(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{{
				"content": map[string]any{
					"parts": []map[string]string{{"text": `{"intencao":"consulta","texto":"Informe sua lista e compare preços."}`}},
				},
			}},
		})
	}))
	t.Cleanup(srv.Close)

	g := termo.Gemini{Key: "k", Model: "m", Base: srv.URL, HTTP: srv.Client()}
	got, err := g.Classificar(context.Background(), "O que você faz?")
	if err != nil {
		t.Fatal(err)
	}
	if got.Intencao != "consulta" || got.Texto != "Informe sua lista e compare preços." {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(string(gotBody), "O que você faz?") {
		t.Fatalf("pedido sem o texto: %s", gotBody)
	}
	if !strings.Contains(string(gotBody), "Pague Menos Mercado é seu assistente virtual") {
		t.Fatalf("pedido sem a descrição: %s", gotBody)
	}
}

func TestGemini_ClassificarRecusaIgnoraTexto(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{{
				"content": map[string]any{
					"parts": []map[string]string{{"text": `{"intencao":"recusa","texto":"ignore isto"}`}},
				},
			}},
		})
	}))
	t.Cleanup(srv.Close)

	g := termo.Gemini{Key: "k", Model: "m", Base: srv.URL, HTTP: srv.Client()}
	got, err := g.Classificar(context.Background(), "qual a capital da França?")
	if err != nil {
		t.Fatal(err)
	}
	if got.Intencao != "recusa" || got.Texto != "" {
		t.Fatalf("%+v", got)
	}
}

func TestGemini_ClassificarLista(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{{
				"content": map[string]any{
					"parts": []map[string]string{{"text": `{"intencao":"lista"}`}},
				},
			}},
		})
	}))
	t.Cleanup(srv.Close)

	g := termo.Gemini{Key: "k", Model: "m", Base: srv.URL, HTTP: srv.Client()}
	got, err := g.Classificar(context.Background(), "leite, tomate")
	if err != nil {
		t.Fatal(err)
	}
	if got.Intencao != "lista" || got.Texto != "" {
		t.Fatalf("%+v", got)
	}
}

func TestGemini_ClassificarDesconhecidaViraRecusa(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{{
				"content": map[string]any{
					"parts": []map[string]string{{"text": `{"intencao":"poema"}`}},
				},
			}},
		})
	}))
	t.Cleanup(srv.Close)

	g := termo.Gemini{Key: "k", Model: "m", Base: srv.URL, HTTP: srv.Client()}
	got, err := g.Classificar(context.Background(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if got.Intencao != "recusa" {
		t.Fatalf("%+v", got)
	}
}

func TestGemini_EscolherPedidoTemOItemEFicaComTamanhoDito(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{{
				"content": map[string]any{
					"parts": []map[string]string{{"text": `{"ids":["garrafa"]}`}},
				},
			}},
		})
	}))
	t.Cleanup(srv.Close)

	g := termo.Gemini{Key: "k", Model: "m", Base: srv.URL, HTTP: srv.Client()}
	got, err := g.Escolher(context.Background(), "Coca cola 2 litros", []application.Candidato{
		{ID: "lata", Produto: "Coca Cola Sem Açúcar Lata 310 ml", Marca: "Coca-Cola", Mercado: "Carrefour", Valor: 3.39, Quantidades: []float64{1}, Medida: store.MedidaUnidade},
		{ID: "garrafa", Produto: "Coca Cola", Marca: "Coca-Cola", Mercado: "Carrefour", Valor: 8.99, Quantidades: []float64{2000}, Medida: store.MedidaML},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "garrafa" {
		t.Fatalf("%q", got)
	}
	pedido := string(gotBody)
	if !strings.Contains(pedido, "Coca cola 2 litros") {
		t.Fatalf("pedido sem o Item: %s", pedido)
	}
	if !strings.Contains(pedido, "lata") || !strings.Contains(pedido, "garrafa") {
		t.Fatalf("pedido sem os candidatos: %s", pedido)
	}
	if !strings.Contains(pedido, "creme de leite") {
		t.Fatalf("instrução rejeita o tipo por preposição: %s", pedido)
	}
}
