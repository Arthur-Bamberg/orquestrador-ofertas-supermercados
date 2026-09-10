package coleta_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/infra/coleta"
)

func TestCliente_ColetarPostaTermoSemMercado(t *testing.T) {
	var gotBody, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/coletas" {
			t.Fatalf("%s %s", r.Method, r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ofertas":[{"id":"o1","produtoId":"leite","mercadoId":"fort","valor":5.5,"quantidades":[1000],"medida":"ml","dataInicio":"2026-09-03","dataExpiracao":"2026-09-03"}]}`))
	}))
	defer srv.Close()

	ofertas, err := coleta.Cliente{Base: srv.URL, Token: "secret"}.Coletar(context.Background(), "leite")
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("auth %q", gotAuth)
	}
	if gotBody != `{"termo":"leite"}` {
		t.Fatalf("body %s", gotBody)
	}
	if len(ofertas) != 1 || ofertas[0].ID != "o1" {
		t.Fatalf("%+v", ofertas)
	}
}

func TestCliente_ColetarErroHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	_, err := coleta.Cliente{Base: srv.URL}.Coletar(context.Background(), "leite")
	if err == nil {
		t.Fatal("expected error")
	}
}
