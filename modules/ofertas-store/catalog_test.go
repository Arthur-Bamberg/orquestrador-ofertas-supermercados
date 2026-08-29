package store_test

import (
	"context"
	"errors"
	"os"
	"testing"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store/storetest"
)

func TestMain(m *testing.M) {
	code := m.Run()
	storetest.Stop()
	os.Exit(code)
}

func TestChaveUnicaOfertaStable(t *testing.T) {
	got := store.ChaveUnicaOferta(store.Oferta{
		ProdutoID: "p1", MercadoID: "m1", Valor: 9.99,
		Quantidades: []float64{500}, Medida: store.MedidaG,
		DataInicio: "2026-07-01", DataExpiracao: "2026-07-10",
	})
	const want = "eab46759feb7d43630072fa68d815308e0820ed6017286bd3dda8076d837cfd5"
	if got != want {
		t.Fatalf("hash changed: got %s want %s", got, want)
	}
}

func TestDeleteGuards(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()

	if err := catalog.SaveMercado(ctx, store.Mercado{ID: "m1", Nome: "Mercado"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveFonte(ctx, store.Fonte{ID: "f1", MercadoID: "m1", URL: "https://example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.DeleteMercado(ctx, "m1"); !errors.Is(err, store.ErrDeleteBlocked) {
		t.Fatalf("DeleteMercado err=%v", err)
	}

	if err := catalog.SaveDocumento(ctx, store.Documento{ID: "d1", FonteID: "f1", MercadoID: "m1", Filename: "a.pdf", Dia: "2026-07-21"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.DeleteFonte(ctx, "f1"); !errors.Is(err, store.ErrDeleteBlocked) {
		t.Fatalf("DeleteFonte err=%v", err)
	}

	if err := catalog.SaveProduto(ctx, store.Produto{ID: "p1", Nome: "Arroz", NomeNorm: "arroz"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveMarca(ctx, store.Marca{ID: "ma1", Nome: "Boa", NomeNorm: "boa"}); err != nil {
		t.Fatal(err)
	}
	marcaID := store.MarcaID("ma1")
	oferta := store.Oferta{
		ID: "o1", ProdutoID: "p1", MarcaID: &marcaID, MercadoID: "m1", Valor: 10,
		Quantidades: []float64{1}, Medida: store.MedidaUnidade, DataInicio: "2026-07-21", DataExpiracao: "2026-07-22",
	}
	if err := catalog.SaveOferta(ctx, oferta, []store.DocumentoID{"d1"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.DeleteProduto(ctx, "p1"); !errors.Is(err, store.ErrDeleteBlocked) {
		t.Fatalf("DeleteProduto err=%v", err)
	}
	if err := catalog.DeleteMarca(ctx, "ma1"); !errors.Is(err, store.ErrDeleteBlocked) {
		t.Fatalf("DeleteMarca err=%v", err)
	}
}

func TestDeleteDocumentoUnlinksAndDeletesOrphanOferta(t *testing.T) {
	catalog := storetest.New(t)
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

	if err := catalog.SaveDocumento(ctx, store.Documento{ID: "d1", FonteID: "f1", MercadoID: "m1", Filename: "a.pdf", Dia: "2026-07-21"}); err != nil {
		t.Fatal(err)
	}
	oferta := store.Oferta{
		ID: "o1", ProdutoID: "p1", MercadoID: "m1", Valor: 10,
		Quantidades: []float64{1}, Medida: store.MedidaUnidade, DataInicio: "2026-07-21", DataExpiracao: "2026-07-22",
	}
	if err := catalog.SaveOferta(ctx, oferta, []store.DocumentoID{"d1"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveFalhasDocumento(ctx, "d1", []store.FalhaExtracao{{Codigo: "x"}}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveUsoExtrator(ctx, store.UsoExtrator{DocumentoID: "d1", Tentativa: "t1", Provider: "stub"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.DeleteDocumento(ctx, "d1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := catalog.GetOferta(ctx, "o1"); err != nil || ok {
		t.Fatalf("orphan oferta ok=%v err=%v", ok, err)
	}
	if falhas, err := catalog.GetFalhasDocumento(ctx, "d1"); err != nil || len(falhas) != 0 {
		t.Fatalf("falhas=%v err=%v", falhas, err)
	}
	if _, ok, err := catalog.GetUsoExtrator(ctx, "d1", "t1"); err != nil || ok {
		t.Fatalf("uso após delete documento ok=%v err=%v", ok, err)
	}
}

func TestCatalogCRUDRoundTrip(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()

	t.Run("Mercado", func(t *testing.T) {
		m := store.Mercado{ID: "m1", Nome: "Fort"}
		if err := catalog.SaveMercado(ctx, m); err != nil {
			t.Fatal(err)
		}
		got, ok, err := catalog.GetMercado(ctx, "m1")
		if err != nil || !ok || got != m {
			t.Fatalf("GetMercado ok=%v err=%v got=%+v", ok, err, got)
		}
		list, err := catalog.ListMercados(ctx)
		if err != nil || len(list) != 1 || list[0].ID != "m1" {
			t.Fatalf("ListMercados=%v err=%v", list, err)
		}
	})

	t.Run("Fonte", func(t *testing.T) {
		f := store.Fonte{ID: "f1", MercadoID: "m1", URL: "https://example.com", FiltroNomeDocumento: "encarte"}
		if err := catalog.SaveFonte(ctx, f); err != nil {
			t.Fatal(err)
		}
		got, ok, err := catalog.GetFonte(ctx, "f1")
		if err != nil || !ok || got != f {
			t.Fatalf("GetFonte ok=%v err=%v got=%+v", ok, err, got)
		}
	})

	t.Run("Produto_NomeNorm", func(t *testing.T) {
		p := store.Produto{ID: "p1", Nome: "Arroz integral", NomeNorm: "arroz integral", Categorias: []string{"mercearia"}}
		if err := catalog.SaveProduto(ctx, p); err != nil {
			t.Fatal(err)
		}
		got, ok, err := catalog.GetProdutoByNomeNorm(ctx, "arroz integral")
		if err != nil || !ok || got.ID != "p1" {
			t.Fatalf("GetProdutoByNomeNorm ok=%v err=%v got=%+v", ok, err, got)
		}
	})

	t.Run("Marca_NomeNorm", func(t *testing.T) {
		m := store.Marca{ID: "ma1", Nome: "Camil", NomeNorm: "camil"}
		if err := catalog.SaveMarca(ctx, m); err != nil {
			t.Fatal(err)
		}
		got, ok, err := catalog.GetMarcaByNomeNorm(ctx, "camil")
		if err != nil || !ok || got.ID != "ma1" {
			t.Fatalf("GetMarcaByNomeNorm ok=%v err=%v got=%+v", ok, err, got)
		}
	})

	t.Run("Documento", func(t *testing.T) {
		d := store.Documento{ID: "d1", FonteID: "f1", MercadoID: "m1", Filename: "a.pdf", Dia: "2026-07-21", Estado: store.EstadoProcessando}
		if err := catalog.SaveDocumento(ctx, d); err != nil {
			t.Fatal(err)
		}
		got, ok, err := catalog.GetDocumento(ctx, "d1")
		if err != nil || !ok || got.Filename != "a.pdf" {
			t.Fatalf("GetDocumento ok=%v err=%v got=%+v", ok, err, got)
		}
		byID, ok, err := catalog.GetDocumentoByIdentity(ctx, "f1", "a.pdf", "2026-07-21")
		if err != nil || !ok || byID.ID != "d1" {
			t.Fatalf("GetDocumentoByIdentity ok=%v err=%v got=%+v", ok, err, byID)
		}
	})

	t.Run("Oferta", func(t *testing.T) {
		o := sampleOferta("o1", "p1", "m1")
		if err := catalog.SaveOferta(ctx, o, []store.DocumentoID{"d1"}); err != nil {
			t.Fatal(err)
		}
		got, ok, err := catalog.GetOferta(ctx, "o1")
		if err != nil || !ok || got.Valor != 10 {
			t.Fatalf("GetOferta ok=%v err=%v got=%+v", ok, err, got)
		}
		list, err := catalog.ListOfertasByDocumento(ctx, "d1")
		if err != nil || len(list) != 1 || list[0].ID != "o1" {
			t.Fatalf("ListOfertasByDocumento=%v err=%v", list, err)
		}
	})
}

func TestSaveOferta_ChaveUnicaConflito(t *testing.T) {
	catalog := storetest.New(t)
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
	if err := catalog.SaveDocumento(ctx, store.Documento{ID: "d1", FonteID: "f1", MercadoID: "m1", Filename: "a.pdf", Dia: "2026-07-21"}); err != nil {
		t.Fatal(err)
	}
	first := sampleOferta("o1", "p1", "m1")
	if err := catalog.SaveOferta(ctx, first, []store.DocumentoID{"d1"}); err != nil {
		t.Fatal(err)
	}
	clone := sampleOferta("o2", "p1", "m1")
	if err := catalog.SaveOferta(ctx, clone, []store.DocumentoID{"d1"}); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("err=%v want ErrConflict", err)
	}
}

func TestUsoExtratorRoundTrip(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	if err := catalog.SaveMercado(ctx, store.Mercado{ID: "m1", Nome: "M"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveFonte(ctx, store.Fonte{ID: "f1", MercadoID: "m1", URL: "https://example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveDocumento(ctx, store.Documento{ID: "d1", FonteID: "f1", MercadoID: "m1", Filename: "a.pdf", Dia: "2026-07-21"}); err != nil {
		t.Fatal(err)
	}
	uso := store.UsoExtrator{
		DocumentoID: "d1", Tentativa: "20260721T080000",
		ArtefatoPath: "artefatos/f1/d1/20260721T080000", Provider: "stub", Model: "stub",
		PromptTokens: 10, OutputTokens: 4,
	}
	if err := catalog.SaveUsoExtrator(ctx, uso); err != nil {
		t.Fatal(err)
	}
	got, ok, err := catalog.GetUsoExtrator(ctx, "d1", "20260721T080000")
	if err != nil || !ok || got.PromptTokens != 10 {
		t.Fatalf("ok=%v err=%v got=%+v", ok, err, got)
	}
	list, err := catalog.ListUsosExtrator(ctx, "d1")
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%v err=%v", list, err)
	}
	if err := catalog.DeleteUsoExtrator(ctx, "d1", "20260721T080000"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := catalog.GetUsoExtrator(ctx, "d1", "20260721T080000"); err != nil || ok {
		t.Fatalf("deleted ok=%v err=%v", ok, err)
	}
}

func TestSaveUsoExtrator_DocumentoAusente(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	err := catalog.SaveUsoExtrator(ctx, store.UsoExtrator{DocumentoID: "d-inexistente", Tentativa: "t1"})
	if err == nil {
		t.Fatal("esperado erro de FK")
	}
	if errors.Is(err, store.ErrDeleteBlocked) {
		t.Fatalf("insert não deve ser ErrDeleteBlocked: %v", err)
	}
}

func TestSaveFonte_MercadoAusenteNaoBloqueiaComoDelete(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	err := catalog.SaveFonte(ctx, store.Fonte{ID: "f1", MercadoID: "inexistente", URL: "https://example.com"})
	if err == nil {
		t.Fatal("esperado erro de FK ao inserir Fonte sem Mercado")
	}
	if errors.Is(err, store.ErrDeleteBlocked) {
		t.Fatalf("insert com Mercado ausente não deve ser ErrDeleteBlocked: %v", err)
	}
}

func TestEarliestDia_SemDocumentos(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	dia, ok, err := catalog.EarliestDia(ctx, "f1", "a.pdf")
	if err != nil || ok || dia != "" {
		t.Fatalf("dia=%q ok=%v err=%v", dia, ok, err)
	}
}

func TestCotaExtrator_PorProviderEDia(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	ok, err := catalog.Esgotado(ctx, "gemini", "2026-07-21")
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if err := catalog.MarcarEsgotado(ctx, "gemini", "2026-07-21"); err != nil {
		t.Fatal(err)
	}
	ok, err = catalog.Esgotado(ctx, "gemini", "2026-07-21")
	if err != nil || !ok {
		t.Fatalf("marcado ok=%v err=%v", ok, err)
	}
	ok, err = catalog.Esgotado(ctx, "gemini", "2026-07-22")
	if err != nil || ok {
		t.Fatalf("outro dia ok=%v err=%v", ok, err)
	}
}

func sampleOferta(id store.OfertaID, produto store.ProdutoID, mercado store.MercadoID) store.Oferta {
	return store.Oferta{
		ID: id, ProdutoID: produto, MercadoID: mercado, Valor: 10,
		Quantidades: []float64{1}, Medida: store.MedidaUnidade,
		DataInicio: "2026-07-21", DataExpiracao: "2026-07-22",
	}
}
