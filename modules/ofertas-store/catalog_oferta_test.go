package store_test

import (
	"context"
	"testing"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store/storetest"
)

func TestSaveOfertasForDocumento_SubstituiIncluindoVazio(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	seedOfertaGraph(t, catalog, "d1")
	docID := store.DocumentoID("d1")
	repo := store.NewOfertaRepo(catalog)

	if err := repo.SaveAll(ctx, docID, []store.Oferta{{
		ID: "o1", DocumentoID: docID, ProdutoID: "p1", MercadoID: "m1", Valor: 1.5,
		Quantidades: []float64{1}, Medida: store.MedidaUnidade,
		DataInicio: "2026-07-01", DataExpiracao: "2026-07-02",
	}}); err != nil {
		t.Fatal(err)
	}
	list, _ := repo.ListByDocumento(ctx, docID)
	if len(list) != 1 {
		t.Fatalf("want 1 got %d", len(list))
	}
	if err := repo.SaveAll(ctx, docID, nil); err != nil {
		t.Fatal(err)
	}
	list, _ = repo.ListByDocumento(ctx, docID)
	if len(list) != 0 {
		t.Fatalf("want empty after replace, got %d", len(list))
	}
}

func TestSaveOfertasForDocumento_IndicePorProduto(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	seedOfertaGraph(t, catalog, "d1")
	if err := catalog.SaveProduto(ctx, store.Produto{ID: "p2", Nome: "P2", NomeNorm: "p2"}); err != nil {
		t.Fatal(err)
	}
	docID := store.DocumentoID("d1")
	repo := store.NewOfertaRepo(catalog)

	if err := repo.SaveAll(ctx, docID, []store.Oferta{
		ofertaComProduto("o1", docID, "p1", 1),
		ofertaComProduto("o2", docID, "p2", 2),
		ofertaComProduto("o3", docID, "p1", 3),
	}); err != nil {
		t.Fatal(err)
	}

	for _, pid := range []store.ProdutoID{"p1", "p2"} {
		ids, err := repo.ListDocumentoIDsByProduto(ctx, pid)
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) != 1 || ids[0] != docID {
			t.Fatalf("produto %s: want [%s], got %v", pid, docID, ids)
		}
	}
}

func TestSaveOfertasForDocumento_AtualizaIndiceNoResave(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	seedOfertaGraph(t, catalog, "d1")
	if err := catalog.SaveProduto(ctx, store.Produto{ID: "p2", Nome: "P2", NomeNorm: "p2"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveProduto(ctx, store.Produto{ID: "p3", Nome: "P3", NomeNorm: "p3"}); err != nil {
		t.Fatal(err)
	}
	docID := store.DocumentoID("d1")
	repo := store.NewOfertaRepo(catalog)

	if err := repo.SaveAll(ctx, docID, []store.Oferta{
		ofertaComProduto("o1", docID, "p1", 1),
		ofertaComProduto("o2", docID, "p2", 2),
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveAll(ctx, docID, []store.Oferta{
		ofertaComProduto("o3", docID, "p2", 3),
		ofertaComProduto("o4", docID, "p3", 4),
	}); err != nil {
		t.Fatal(err)
	}

	assertDocIDs(t, repo, "p1", nil)
	assertDocIDs(t, repo, "p2", []store.DocumentoID{docID})
	assertDocIDs(t, repo, "p3", []store.DocumentoID{docID})
}

func TestSaveOfertasForDocumento_UnicaEntreDocumentos(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	seedOfertaGraph(t, catalog, "d1")
	if err := catalog.SaveDocumento(ctx, store.Documento{ID: "d2", FonteID: "f1", MercadoID: "m1", Filename: "b.pdf", Dia: "2026-07-22"}); err != nil {
		t.Fatal(err)
	}
	repo := store.NewOfertaRepo(catalog)
	o := store.Oferta{
		ID: "o1", ProdutoID: "p1", MercadoID: "m1", Valor: 5,
		Quantidades: []float64{1}, Medida: store.MedidaUnidade,
		DataInicio: "2026-07-01", DataExpiracao: "2026-07-02",
	}
	if err := repo.SaveAll(ctx, "d1", []store.Oferta{o}); err != nil {
		t.Fatal(err)
	}
	clone := o
	clone.ID = "o2"
	if err := repo.SaveAll(ctx, "d2", []store.Oferta{clone}); err != nil {
		t.Fatal(err)
	}
	list1, _ := repo.ListByDocumento(ctx, "d1")
	list2, _ := repo.ListByDocumento(ctx, "d2")
	if len(list1) != 1 || len(list2) != 1 {
		t.Fatalf("list1=%d list2=%d", len(list1), len(list2))
	}
	if list1[0].ID != list2[0].ID {
		t.Fatalf("want same id, got %s vs %s", list1[0].ID, list2[0].ID)
	}
	if err := repo.SaveAll(ctx, "d1", nil); err != nil {
		t.Fatal(err)
	}
	list2, _ = repo.ListByDocumento(ctx, "d2")
	if len(list2) != 1 {
		t.Fatalf("d2 should keep oferta, got %d", len(list2))
	}
	if err := repo.SaveAll(ctx, "d2", nil); err != nil {
		t.Fatal(err)
	}
	_, ok, err := repo.GetByUniq(ctx, store.ChaveUnicaOferta(o))
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("orphan oferta should be deleted")
	}
}

func TestEarliestDia_MenorDiaDaFonteEFilename(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	if err := catalog.SaveMercado(ctx, store.Mercado{ID: "m1", Nome: "Fort"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveFonte(ctx, store.Fonte{ID: "fonte-fort", MercadoID: "m1", URL: "https://example.com"}); err != nil {
		t.Fatal(err)
	}
	repo := store.NewDocumentoRepo(catalog)
	fonte := store.FonteID("fonte-fort")
	file := "encarte.pdf"
	if err := repo.Save(ctx, store.Documento{
		ID: "d2", FonteID: fonte, MercadoID: "m1", Filename: file, Dia: "2026-07-18", Estado: store.EstadoConcluido,
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, store.Documento{
		ID: "d1", FonteID: fonte, MercadoID: "m1", Filename: file, Dia: "2026-07-10", Estado: store.EstadoFalhou,
	}); err != nil {
		t.Fatal(err)
	}
	dia, ok, err := repo.EarliestDia(ctx, fonte, file)
	if err != nil || !ok || dia != "2026-07-10" {
		t.Fatalf("earliest=%q ok=%v err=%v", dia, ok, err)
	}
}

func seedOfertaGraph(t *testing.T, catalog *store.Catalog, documentoID store.DocumentoID) {
	t.Helper()
	ctx := context.Background()
	if err := catalog.SaveMercado(ctx, store.Mercado{ID: "m1", Nome: "M"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveFonte(ctx, store.Fonte{ID: "f1", MercadoID: "m1", URL: "https://example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveProduto(ctx, store.Produto{ID: "p1", Nome: "P", NomeNorm: "p"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveDocumento(ctx, store.Documento{ID: documentoID, FonteID: "f1", MercadoID: "m1", Filename: "a.pdf", Dia: "2026-07-21"}); err != nil {
		t.Fatal(err)
	}
}

func ofertaComProduto(id store.OfertaID, doc store.DocumentoID, produto store.ProdutoID, valor float64) store.Oferta {
	return store.Oferta{
		ID: id, DocumentoID: doc, ProdutoID: produto, MercadoID: "m1", Valor: valor,
		Quantidades: []float64{1}, Medida: store.MedidaUnidade,
		DataInicio: "2026-07-01", DataExpiracao: "2026-07-02",
	}
}

func assertDocIDs(t *testing.T, repo *store.OfertaRepo, produtoID store.ProdutoID, want []store.DocumentoID) {
	t.Helper()
	got, err := repo.ListDocumentoIDsByProduto(context.Background(), produtoID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("produto %s: want %v, got %v", produtoID, want, got)
	}
	set := map[store.DocumentoID]struct{}{}
	for _, id := range got {
		set[id] = struct{}{}
	}
	for _, id := range want {
		if _, ok := set[id]; !ok {
			t.Fatalf("produto %s: missing %s in %v", produtoID, id, got)
		}
	}
}
