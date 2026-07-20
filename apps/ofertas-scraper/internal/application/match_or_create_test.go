package application_test

import (
	"context"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

func TestPersistirOfertasValidas_MatchOrCreateAndCategoriasUnion(t *testing.T) {
	produtos := &memProdutos{byNorm: map[string]domain.Produto{
		"arroz integral": {
			ID: "p1", Nome: "Arroz integral", NomeNorm: "arroz integral",
			Categorias: []string{"mercearia"},
		},
	}}
	marcas := &memMarcas{}
	doc := domain.Documento{ID: "d1", MercadoID: "m1"}

	ofertas, err := application.PersistirOfertasValidas(context.Background(), produtos, marcas, doc, []domain.OfertaValidada{
		{
			Produto: "Arroz Integral", Marca: "Camil", Categorias: []string{"grãos"},
			Valor: 10, Quantidades: []float64{1000}, Medida: domain.MedidaG,
			DataInicio: "2026-07-18", DataExpiracao: "2026-07-25",
			OrigemDataInicio: domain.OrigemExtrator, OrigemDataExpiracao: domain.OrigemExtrator,
			Comparativo: &domain.Comparativo{Quantidade: 200, Valor: 2},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ofertas) != 1 {
		t.Fatalf("ofertas=%d", len(ofertas))
	}
	if ofertas[0].ProdutoID != "p1" {
		t.Fatalf("produtoId=%s want p1", ofertas[0].ProdutoID)
	}
	if ofertas[0].MarcaID == nil {
		t.Fatal("marcaId nil")
	}
	if ofertas[0].Comparativo == nil || ofertas[0].Comparativo.Quantidade != 200 {
		t.Fatalf("comparativo not persisted: %#v", ofertas[0].Comparativo)
	}
	p := produtos.byNorm["arroz integral"]
	if len(p.Categorias) != 2 {
		t.Fatalf("categorias=%v", p.Categorias)
	}
	if _, ok := marcas.byNorm["camil"]; !ok {
		t.Fatal("marca not created")
	}
}
