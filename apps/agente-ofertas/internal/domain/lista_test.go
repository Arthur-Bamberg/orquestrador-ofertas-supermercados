package domain_test

import (
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/domain"
)

func TestParseLista_separaItensPorVirgulaQuebraEConectivoE(t *testing.T) {
	got := domain.ParseLista("leite, arroz e café\nfeijão")
	if len(got.Itens) != 4 {
		t.Fatalf("itens=%d %+v", len(got.Itens), got.Itens)
	}
	want := []string{"leite", "arroz", "café", "feijão"}
	for i, w := range want {
		if got.Itens[i].Texto != w {
			t.Fatalf("item[%d]=%q want %q", i, got.Itens[i].Texto, w)
		}
	}
}

func TestParseLista_textoVazioNaoTemItens(t *testing.T) {
	got := domain.ParseLista("   \n  ")
	if len(got.Itens) != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestTermoDoItem_tiraInvólucroEMantemTipoDito(t *testing.T) {
	casos := []struct{ item, want string }{
		{"Comprar: cenoura", "cenoura"},
		{"Quero 2kg de cenoura orgânica, por favor", "2kg cenoura orgânica"},
		{"Creme de leite", "creme de leite"},
		{"molho de tomate", "molho de tomate"},
		{"Coca cola 2 litros", "coca cola 2 litros"},
		{"abobrinha", "abobrinha"},
		{"leite Italac", "leite italac"},
		{"Comprar:", ""},
		{"quero 2kg", ""},
	}
	for _, c := range casos {
		if got := domain.TermoDoItem(c.item); got != c.want {
			t.Fatalf("TermoDoItem(%q)=%q want %q", c.item, got, c.want)
		}
	}
}

func TestTipoDoTermo_tiraTamanhoEMantemTipo(t *testing.T) {
	if got := domain.TipoDoTermo("coca cola 2 litros"); got != "coca cola" {
		t.Fatalf("%q", got)
	}
}
