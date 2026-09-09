package application_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store/storetest"
)

func TestMain(m *testing.M) {
	code := m.Run()
	storetest.Stop()
	os.Exit(code)
}

type fakeVitrine struct {
	cartoes  []domain.CartaoVitrine
	esgotada bool
	err      error
	calls    *int
}

func (f fakeVitrine) Buscar(context.Context, string) (domain.ResultadoVitrine, error) {
	if f.calls != nil {
		*f.calls++
	}
	if f.err != nil {
		return domain.ResultadoVitrine{}, f.err
	}
	return domain.ResultadoVitrine{Cartoes: f.cartoes, Esgotada: f.esgotada}, nil
}

func TestColetar_PersisteTodoCartaoCompletoInclusiveRelacionado(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	if err := catalog.SaveMercado(ctx, store.Mercado{ID: "mercado-fort", Nome: "Fort Atacadista"}); err != nil {
		t.Fatal(err)
	}
	vitrines := map[store.MercadoID]domain.Vitrine{
		"mercado-fort": fakeVitrine{esgotada: true, cartoes: []domain.CartaoVitrine{
			{Nome: "Tomate Carmem 1kg", Marca: "Carmem", Valor: 6.9, Quantidades: []float64{1000}, Medida: store.MedidaG, IndicacaoPromocional: true},
			{Nome: "Molho de tomate 340g", Marca: "Quero", Valor: 3.5, Quantidades: []float64{340}, Medida: store.MedidaG, IndicacaoPromocional: false},
			{Nome: "Tomate italiano sem preço", Marca: "Carmem", Valor: 0, Quantidades: []float64{1000}, Medida: store.MedidaG},
		}},
	}
	got, err := application.Coletar(ctx, catalog, vitrines, "tomate", "2026-09-07")
	if err != nil {
		t.Fatal(err)
	}
	coleta, ok, err := catalog.GetColetaByIdentity(ctx, "tomate", "mercado-fort", "2026-09-07")
	if err != nil || !ok {
		t.Fatalf("coleta ok=%v err=%v", ok, err)
	}
	if coleta.Estado != store.EstadoParcial {
		t.Fatalf("estado=%s want parcial (1 incomplete card)", coleta.Estado)
	}
	list, err := store.NewOfertaRepo(catalog).ListByColeta(ctx, coleta.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 persisted cards, got %d", len(list))
	}
	if len(got) != 2 {
		t.Fatalf("return want 2 Ofertas, got %d", len(got))
	}
	produtos := map[store.ProdutoID]struct{}{}
	byValor := map[float64]store.Oferta{}
	for _, o := range list {
		produtos[o.ProdutoID] = struct{}{}
		byValor[o.Valor] = o
		if o.DataInicio != "2026-09-07" || o.OrigemDataInicio != store.OrigemColeta {
			t.Fatalf("vigência/origem %#v", o)
		}
		if o.ProdutoID == "" {
			t.Fatal("Oferta must point at a match-or-created Produto")
		}
	}
	if len(produtos) != 2 {
		t.Fatalf("related card must be its own Produto, got %d", len(produtos))
	}
	if !byValor[6.9].IndicacaoPromocional || byValor[3.5].IndicacaoPromocional {
		t.Fatalf("indicação from card: 6.9=%v 3.5=%v", byValor[6.9].IndicacaoPromocional, byValor[3.5].IndicacaoPromocional)
	}
	tomate, ok, err := catalog.GetProduto(ctx, byValor[6.9].ProdutoID)
	if err != nil || !ok {
		t.Fatalf("tomate produto ok=%v err=%v", ok, err)
	}
	if tomate.NomeNorm == "tomate" {
		t.Fatalf("must not force the search term as Produto, got %q", tomate.Nome)
	}
}

func TestColetar_ReusaConcluidoSemBuscarDeNovo(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	if err := catalog.SaveMercado(ctx, store.Mercado{ID: "mercado-fort", Nome: "Fort Atacadista"}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	vitrines := map[store.MercadoID]domain.Vitrine{
		"mercado-fort": fakeVitrine{calls: &calls, esgotada: true, cartoes: []domain.CartaoVitrine{
			{Nome: "Tomate 1kg", Valor: 6.9, Quantidades: []float64{1000}, Medida: store.MedidaG},
		}},
	}
	if _, err := application.Coletar(ctx, catalog, vitrines, "tomate", "2026-09-07"); err != nil {
		t.Fatal(err)
	}
	second := map[store.MercadoID]domain.Vitrine{
		"mercado-fort": fakeVitrine{calls: &calls, esgotada: true, cartoes: []domain.CartaoVitrine{
			{Nome: "Tomate 1kg", Valor: 5.5, Quantidades: []float64{1000}, Medida: store.MedidaG},
		}},
	}
	got, err := application.Coletar(ctx, catalog, second, "tomate", "2026-09-07")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("want 1 site hit, got %d", calls)
	}
	if len(got) != 1 || got[0].Valor != 6.9 {
		t.Fatalf("reuse want 6.9, got %#v", got)
	}
}

func TestColetar_RetentaFalhouESubstituiConjunto(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	if err := catalog.SaveMercado(ctx, store.Mercado{ID: "mercado-fort", Nome: "Fort Atacadista"}); err != nil {
		t.Fatal(err)
	}
	first := map[store.MercadoID]domain.Vitrine{
		"mercado-fort": fakeVitrine{err: errors.New("fort down")},
	}
	if _, err := application.Coletar(ctx, catalog, first, "tomate", "2026-09-07"); err != nil {
		t.Fatal(err)
	}
	coleta, ok, err := catalog.GetColetaByIdentity(ctx, "tomate", "mercado-fort", "2026-09-07")
	if err != nil || !ok || coleta.Estado != store.EstadoFalhou {
		t.Fatalf("want falhou, got ok=%v %#v err=%v", ok, coleta, err)
	}
	second := map[store.MercadoID]domain.Vitrine{
		"mercado-fort": fakeVitrine{esgotada: true, cartoes: []domain.CartaoVitrine{
			{Nome: "Tomate 1kg", Valor: 5.5, Quantidades: []float64{1000}, Medida: store.MedidaG},
		}},
	}
	got, err := application.Coletar(ctx, catalog, second, "tomate", "2026-09-07")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Valor != 5.5 {
		t.Fatalf("retry want 5.5, got %#v", got)
	}
}

func TestColetar_PesquisaCortadaComOfertasFicaParcial(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	if err := catalog.SaveMercado(ctx, store.Mercado{ID: "mercado-fort", Nome: "Fort Atacadista"}); err != nil {
		t.Fatal(err)
	}
	vitrines := map[store.MercadoID]domain.Vitrine{
		"mercado-fort": fakeVitrine{esgotada: false, cartoes: []domain.CartaoVitrine{
			{Nome: "Tomate 1kg", Valor: 6.9, Quantidades: []float64{1000}, Medida: store.MedidaG},
		}},
	}
	got, err := application.Coletar(ctx, catalog, vitrines, "tomate", "2026-09-07")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %#v", got)
	}
	coleta, ok, err := catalog.GetColetaByIdentity(ctx, "tomate", "mercado-fort", "2026-09-07")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if coleta.Estado != store.EstadoParcial {
		t.Fatalf("estado=%s want parcial", coleta.Estado)
	}
}

func TestColetar_VitrineVaziaEsgotadaFicaConcluido(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	if err := catalog.SaveMercado(ctx, store.Mercado{ID: "mercado-fort", Nome: "Fort Atacadista"}); err != nil {
		t.Fatal(err)
	}
	vitrines := map[store.MercadoID]domain.Vitrine{
		"mercado-fort": fakeVitrine{esgotada: true, cartoes: nil},
	}
	got, err := application.Coletar(ctx, catalog, vitrines, "xyzzy", "2026-09-07")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
	coleta, ok, err := catalog.GetColetaByIdentity(ctx, "xyzzy", "mercado-fort", "2026-09-07")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if coleta.Estado != store.EstadoConcluido {
		t.Fatalf("estado=%s want concluido", coleta.Estado)
	}
}
