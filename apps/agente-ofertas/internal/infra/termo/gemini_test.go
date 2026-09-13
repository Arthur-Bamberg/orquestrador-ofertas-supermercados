package termo_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/infra/termo"
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
