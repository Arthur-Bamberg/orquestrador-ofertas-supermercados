package application_test

import (
	"context"
	"os"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store/storetest"
)

func TestMain(m *testing.M) {
	code := m.Run()
	storetest.Stop()
	os.Exit(code)
}

type fakeVitrine struct {
	cartoes []domain.CartaoVitrine
}

func (f fakeVitrine) Buscar(context.Context, string) ([]domain.CartaoVitrine, error) {
	return f.cartoes, nil
}

func TestColetarProduto_PersisteSoCartoesQueCasamComIndicacaoDoSite(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	if err := catalog.SaveMercado(ctx, store.Mercado{ID: "mercado-fort", Nome: "Fort Atacadista"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveProduto(ctx, store.Produto{ID: "p1", Nome: "Arroz integral", NomeNorm: "arroz integral"}); err != nil {
		t.Fatal(err)
	}
	vitrines := map[store.MercadoID]domain.Vitrine{
		"mercado-fort": fakeVitrine{cartoes: []domain.CartaoVitrine{
			{Nome: "Arroz Integral Camil 1kg", Marca: "Camil", Valor: 8.9, Quantidades: []float64{1000}, Medida: store.MedidaG, IndicacaoPromocional: true},
			{Nome: "Arroz branco 1kg", Marca: "Camil", Valor: 6, Quantidades: []float64{1000}, Medida: store.MedidaG, IndicacaoPromocional: false},
			{Nome: "Arroz Integral Tio João 5kg", Marca: "Tio João", Valor: 32, Quantidades: []float64{5000}, Medida: store.MedidaG, IndicacaoPromocional: false},
		}},
	}
	if err := application.ColetarProduto(ctx, catalog, vitrines, "p1", "2026-09-05"); err != nil {
		t.Fatal(err)
	}
	coleta, ok, err := catalog.GetColetaByIdentity(ctx, "p1", "mercado-fort", "2026-09-05")
	if err != nil || !ok {
		t.Fatalf("coleta ok=%v err=%v", ok, err)
	}
	if coleta.Estado != store.EstadoConcluido {
		t.Fatalf("estado=%s", coleta.Estado)
	}
	list, err := store.NewOfertaRepo(catalog).ListByColeta(ctx, coleta.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 matching cards, got %d", len(list))
	}
	byValor := map[float64]store.Oferta{}
	for _, o := range list {
		byValor[o.Valor] = o
		if o.DataInicio != "2026-09-05" || o.OrigemDataInicio != store.OrigemColeta {
			t.Fatalf("vigência/origem %#v", o)
		}
		if o.ProdutoID != "p1" {
			t.Fatalf("forced searched produto, got %s", o.ProdutoID)
		}
	}
	if !byValor[8.9].IndicacaoPromocional || byValor[32].IndicacaoPromocional {
		t.Fatalf("indicação from site: 8.9=%v 32=%v", byValor[8.9].IndicacaoPromocional, byValor[32].IndicacaoPromocional)
	}
}

func TestColetarProduto_SemCartaoQueCasaFalhaAColeta(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	if err := catalog.SaveMercado(ctx, store.Mercado{ID: "mercado-asun", Nome: "Asun"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveProduto(ctx, store.Produto{ID: "p1", Nome: "Arroz integral", NomeNorm: "arroz integral"}); err != nil {
		t.Fatal(err)
	}
	vitrines := map[store.MercadoID]domain.Vitrine{
		"mercado-asun": fakeVitrine{cartoes: []domain.CartaoVitrine{
			{Nome: "Feijão preto 1kg", Valor: 8, Quantidades: []float64{1000}, Medida: store.MedidaG},
		}},
	}
	if err := application.ColetarProduto(ctx, catalog, vitrines, "p1", "2026-09-05"); err != nil {
		t.Fatal(err)
	}
	coleta, ok, err := catalog.GetColetaByIdentity(ctx, "p1", "mercado-asun", "2026-09-05")
	if err != nil || !ok {
		t.Fatalf("coleta ok=%v err=%v", ok, err)
	}
	if coleta.Estado != store.EstadoFalhou {
		t.Fatalf("estado=%s", coleta.Estado)
	}
	list, _ := store.NewOfertaRepo(catalog).ListByColeta(ctx, coleta.ID)
	if len(list) != 0 {
		t.Fatalf("ofertas=%d", len(list))
	}
}
