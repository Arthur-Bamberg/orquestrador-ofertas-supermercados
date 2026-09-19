package domain_test

import (
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestProdutoDoCartao_TiraMarcaDoRotulo(t *testing.T) {
	got := domain.ProdutoDoCartao("Tomate Carmem 1kg", "Carmem")
	if got != "Tomate" {
		t.Fatalf("got %q", got)
	}
}

func TestProdutoDoCartao_MarcaAusenteNoRotulo(t *testing.T) {
	got := domain.ProdutoDoCartao("Molho de tomate 340g", "Quero")
	if got != "Molho de tomate" {
		t.Fatalf("got %q", got)
	}
}

func TestProdutoDoCartao_TiraMarcaETamanhoDoRotulo(t *testing.T) {
	tests := []struct {
		name        string
		inputName   string
		inputMarca  string
		expected    string
	}{
		{
			name:      "Tomate Carmem 1kg",
			inputName:   "Tomate Carmem 1kg",
			inputMarca:  "Carmem",
			expected:    "Tomate",
		},
		{
			name:      "Molho de tomate Quero 340g",
			inputName:   "Molho de tomate Quero 340g",
			inputMarca:  "Quero",
			expected:    "Molho de tomate",
		},
		{
			name:      "Creme de Leite Zero Lactose Nestlé 200g",
			inputName:   "Creme de Leite Zero Lactose Nestlé 200g",
			inputMarca:  "Nestlé",
			expected:    "Creme de Leite Zero Lactose",
		},
		{
			name:      "Leite Condensado Mococa Caixa 395g",
			inputName:   "Leite Condensado Mococa Caixa 395g",
			inputMarca:  "Mococa",
			expected:    "Leite Condensado",
		},
		{
			name:      "Água Sanitária Ypê 2L",
			inputName:   "Água Sanitária Ypê 2L",
			inputMarca:  "Ypê",
			expected:    "Água Sanitária",
		},
		{
			name:      "Refrigerante Coca-Cola 350ml",
			inputName:   "Refrigerante Coca-Cola 350ml",
			inputMarca:  "Coca-Cola",
			expected:    "Refrigerante",
		},
		{
			name:      "Sabão em Pó Omo Lavagem Perfeita 1.6kg",
			inputName:   "Sabão em Pó Omo Lavagem Perfeita 1.6kg",
			inputMarca:  "Omo",
			expected:    "Sabão em Pó Lavagem Perfeita",
		},
		{
			name:      "Cerveja Skol Lata 350 ml",
			inputName:   "Cerveja Skol Lata 350 ml",
			inputMarca:  "Skol",
			expected:    "Cerveja",
		},
		{
			name:      "Café Pilão Pouch 500 g",
			inputName:   "Café Pilão Pouch 500 g",
			inputMarca:  "Pilão",
			expected:    "Café",
		},
		{
			name:      "Massa de Tomate Quero Sachê 140g",
			inputName:   "Massa de Tomate Quero Sachê 140g",
			inputMarca:  "Quero",
			expected:    "Massa de Tomate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.ProdutoDoCartao(tt.inputName, tt.inputMarca)
			if got != tt.expected {
				t.Errorf("ProdutoDoCartao(%q, %q) got %q, want %q", tt.inputName, tt.inputMarca, got, tt.expected)
			}
		})
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
