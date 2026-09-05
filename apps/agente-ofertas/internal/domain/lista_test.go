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
