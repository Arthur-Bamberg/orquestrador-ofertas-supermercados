package domain_test

import (
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestProdutoDoCartao_TiraMarcaDoRotulo(t *testing.T) {
	got := domain.ProdutoDoCartao("Tomate Carmem 1kg", "Carmem")
	if got != "Tomate 1kg" {
		t.Fatalf("got %q", got)
	}
}

func TestProdutoDoCartao_MarcaAusenteNoRotulo(t *testing.T) {
	got := domain.ProdutoDoCartao("Molho de tomate 340g", "Quero")
	if got != "Molho de tomate 340g" {
		t.Fatalf("got %q", got)
	}
}

func TestCartaoCompleto(t *testing.T) {
	ok := domain.CartaoVitrine{Valor: 1, Quantidades: []float64{1}, Medida: store.MedidaUnidade}
	if !domain.CartaoCompleto(ok) {
		t.Fatal("complete card")
	}
	if domain.CartaoCompleto(domain.CartaoVitrine{Valor: 0, Quantidades: []float64{1}, Medida: store.MedidaG}) {
		t.Fatal("zero price")
	}
}
