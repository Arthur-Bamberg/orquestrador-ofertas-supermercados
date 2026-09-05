package store_test

import (
	"context"
	"errors"
	"testing"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store/storetest"
)

func TestSaveOfertasForDocumento_IndicacaoPromocionalSempreVerdadeira(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	seedOfertaGraph(t, catalog, "d1")
	repo := store.NewOfertaRepo(catalog)

	if err := repo.SaveAll(ctx, "d1", []store.Oferta{{
		ID: "o1", ProdutoID: "p1", MercadoID: "m1", Valor: 1.5,
		Quantidades: []float64{1}, Medida: store.MedidaUnidade,
		DataInicio: "2026-07-01", DataExpiracao: "2026-07-02",
	}}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := catalog.GetOferta(ctx, "o1")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if !got.IndicacaoPromocional {
		t.Fatal("encarte Oferta must have IndicacaoPromocional true")
	}
}

func TestSaveOferta_ExigeDocumentoOuColeta(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	seedOfertaGraph(t, catalog, "d1")
	o := store.Oferta{
		ID: "o1", ProdutoID: "p1", MercadoID: "m1", Valor: 10,
		Quantidades: []float64{1}, Medida: store.MedidaUnidade,
		DataInicio: "2026-07-01", DataExpiracao: "2026-07-02",
	}
	if err := catalog.SaveOferta(ctx, o, nil); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func TestSaveOfertasForColeta_ReencontroAtualizaIndicacaoDoSite(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	seedOfertaGraph(t, catalog, "d1")
	if err := catalog.SaveColeta(ctx, store.Coleta{
		ID: "c1", ProdutoID: "p1", MercadoID: "m1", Dia: "2026-09-05", Estado: store.EstadoProcessando,
	}); err != nil {
		t.Fatal(err)
	}
	repo := store.NewOfertaRepo(catalog)
	first := store.Oferta{
		ID: "o-site", ProdutoID: "p1", MercadoID: "m1", Valor: 8.9,
		Quantidades: []float64{1000}, Medida: store.MedidaG,
		DataInicio: "2026-09-05", DataExpiracao: "2026-09-05",
		OrigemDataInicio: store.OrigemColeta, OrigemDataExpiracao: store.OrigemColeta,
		IndicacaoPromocional: true,
	}
	if err := repo.SaveAllForColeta(ctx, "c1", []store.Oferta{first}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveAllForColeta(ctx, "c1", []store.Oferta{{
		ID: "o-site-recoleta", ProdutoID: "p1", MercadoID: "m1", Valor: 8.9,
		Quantidades: []float64{1000}, Medida: store.MedidaG,
		DataInicio: "2026-09-05", DataExpiracao: "2026-09-05",
		OrigemDataInicio: store.OrigemColeta, OrigemDataExpiracao: store.OrigemColeta,
		IndicacaoPromocional: false,
	}}); err != nil {
		t.Fatal(err)
	}
	list, err := repo.ListByColeta(ctx, "c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 oferta, got %#v", list)
	}
	if list[0].ID != "o-site" {
		t.Fatalf("same chave reuses existing id, got %s", list[0].ID)
	}
	if list[0].IndicacaoPromocional {
		t.Fatal("site Coleta must ascertain IndicacaoPromocional from the card")
	}
}

func TestSaveOfertasForColeta_ColisaoComEncarteMantemIndicacao(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	seedOfertaGraph(t, catalog, "d1")
	encarte := store.Oferta{
		ID: "o-encarte", ProdutoID: "p1", MercadoID: "m1", Valor: 8.9,
		Quantidades: []float64{1000}, Medida: store.MedidaG,
		DataInicio: "2026-09-05", DataExpiracao: "2026-09-05",
	}
	if err := catalog.SaveOferta(ctx, encarte, []store.DocumentoID{"d1"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveColeta(ctx, store.Coleta{
		ID: "c1", ProdutoID: "p1", MercadoID: "m1", Dia: "2026-09-05", Estado: store.EstadoProcessando,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.NewOfertaRepo(catalog).SaveAllForColeta(ctx, "c1", []store.Oferta{{
		ID: "o-site", ProdutoID: "p1", MercadoID: "m1", Valor: 8.9,
		Quantidades: []float64{1000}, Medida: store.MedidaG,
		DataInicio: "2026-09-05", DataExpiracao: "2026-09-05",
		OrigemDataInicio: store.OrigemColeta, OrigemDataExpiracao: store.OrigemColeta,
		IndicacaoPromocional: false,
	}}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := catalog.GetOferta(ctx, "o-encarte")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if !got.IndicacaoPromocional {
		t.Fatal("Oferta still linked to Documento keeps IndicacaoPromocional true")
	}
}

func TestSaveOfertasForColeta_SubstituiEMarcaIndicacaoDoSite(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	seedOfertaGraph(t, catalog, "d1")
	if err := catalog.SaveColeta(ctx, store.Coleta{
		ID: "c1", ProdutoID: "p1", MercadoID: "m1", Dia: "2026-09-05", Estado: store.EstadoProcessando,
	}); err != nil {
		t.Fatal(err)
	}
	repo := store.NewOfertaRepo(catalog)
	if err := repo.SaveAllForColeta(ctx, "c1", []store.Oferta{{
		ID: "o-site", ProdutoID: "p1", MercadoID: "m1", Valor: 8.9,
		Quantidades: []float64{1000}, Medida: store.MedidaG,
		DataInicio: "2026-09-05", DataExpiracao: "2026-09-05",
		OrigemDataInicio: store.OrigemColeta, OrigemDataExpiracao: store.OrigemColeta,
		IndicacaoPromocional: true,
	}}); err != nil {
		t.Fatal(err)
	}
	list, err := repo.ListByColeta(ctx, "c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || !list[0].IndicacaoPromocional || list[0].Valor != 8.9 {
		t.Fatalf("got %#v", list)
	}
	if err := repo.SaveAllForColeta(ctx, "c1", []store.Oferta{{
		ID: "o-site-2", ProdutoID: "p1", MercadoID: "m1", Valor: 9.5,
		Quantidades: []float64{1000}, Medida: store.MedidaG,
		DataInicio: "2026-09-05", DataExpiracao: "2026-09-05",
		OrigemDataInicio: store.OrigemColeta, OrigemDataExpiracao: store.OrigemColeta,
		IndicacaoPromocional: false,
	}}); err != nil {
		t.Fatal(err)
	}
	list, err = repo.ListByColeta(ctx, "c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].IndicacaoPromocional || list[0].Valor != 9.5 {
		t.Fatalf("replace want 9.5 sem indicação, got %#v", list)
	}
	_, ok, err := catalog.GetOferta(ctx, "o-site")
	if err != nil || ok {
		t.Fatalf("orphan from previous coleta should be gone ok=%v err=%v", ok, err)
	}
}

func TestDeleteDocumento_PreservaOfertaComColeta(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	seedOfertaGraph(t, catalog, "d1")
	o := store.Oferta{
		ID: "o1", ProdutoID: "p1", MercadoID: "m1", Valor: 10,
		Quantidades: []float64{1}, Medida: store.MedidaUnidade,
		DataInicio: "2026-07-21", DataExpiracao: "2026-07-22",
	}
	if err := catalog.SaveOferta(ctx, o, []store.DocumentoID{"d1"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveColeta(ctx, store.Coleta{
		ID: "c1", ProdutoID: "p1", MercadoID: "m1", Dia: "2026-09-05",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.NewOfertaRepo(catalog).SaveAllForColeta(ctx, "c1", []store.Oferta{o}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.DeleteDocumento(ctx, "d1"); err != nil {
		t.Fatal(err)
	}
	got, ok, err := catalog.GetOferta(ctx, "o1")
	if err != nil || !ok {
		t.Fatalf("oferta should remain via Coleta ok=%v err=%v", ok, err)
	}
	if !got.IndicacaoPromocional {
		t.Fatal("encarte association forced IndicacaoPromocional true")
	}
}
