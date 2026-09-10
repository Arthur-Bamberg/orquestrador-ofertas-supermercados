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

func TestGemini_TermoVazioQuandoJSONVazio(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{{
				"content": map[string]any{
					"parts": []map[string]string{{"text": `{"termo":""}`}},
				},
			}},
		})
	}))
	t.Cleanup(srv.Close)

	g := termo.Gemini{Key: "k", Model: "m", Base: srv.URL, HTTP: srv.Client()}
	got, err := g.Termo(context.Background(), "Comprar:")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("%q", got)
	}
}
