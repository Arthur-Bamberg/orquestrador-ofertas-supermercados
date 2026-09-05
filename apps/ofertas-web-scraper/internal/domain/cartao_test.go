package domain_test

import (
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/domain"
)

func TestCartaoCasaProduto_NomeDoProdutoNoRotulo(t *testing.T) {
	if !domain.CartaoCasaProduto("Arroz Integral Camil 1kg", "arroz integral") {
		t.Fatal("marca e tamanho no cartão ainda casam o tipo vendável")
	}
}

func TestCartaoCasaProduto_OutroTipoVendavel(t *testing.T) {
	if domain.CartaoCasaProduto("Arroz branco parboilizado 1kg", "arroz integral") {
		t.Fatal("arroz branco não casa arroz integral")
	}
}

func TestCartaoCasaProduto_Vazio(t *testing.T) {
	if domain.CartaoCasaProduto("Arroz integral", "") || domain.CartaoCasaProduto("", "arroz integral") {
		t.Fatal("rótulo ou produto vazio não casa")
	}
}
