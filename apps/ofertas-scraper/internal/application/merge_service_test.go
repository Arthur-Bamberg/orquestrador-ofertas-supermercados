package application_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/application"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
	storetest "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store/storetest"
)

func TestMergeService_SanitizeProductNamesAndBackfillOffers(t *testing.T) {
	catalog := storetest.New(t)
	service := application.NewMergeService(catalog)

	// Seed initial data
	prodID1 := store.ProdutoID("prod1")
	prodID2 := store.ProdutoID("prod2")
	prodID3 := store.ProdutoID("prod3")
	mercadoID := store.MercadoID("merc1")
	marcaID := store.MarcaID("marca1")
	docID := store.DocumentoID("doc1")
	coletaID := store.ColetaID("coleta1")

	// Create a product that needs sanitization and backfill
	prod1 := store.Produto{
		ID:         prodID1,
		Nome:       "Creme de Leite 200g (Marca X)",
		NomeNorm:   store.NormalizarRotulo("creme de leite 200g (marca x)"),
		Categorias: []string{"laticinios"},
	}
	require.NoError(t, catalog.SaveProduto(ctx, prod1))

	// Create an offer associated with prod1 that needs backfill
	oferta1 := store.Oferta{
		ID:                   store.OfertaID("oferta1"),
		ProdutoID:            prodID1,
		MarcaID:              &marcaID,
		MercadoID:            mercadoID,
		Valor:                3.49,
		Quantidades:          []float64{1},
		Medida:               store.MedidaUnidade,
		DataInicio:           "2026-09-01",
		DataExpiracao:        "2026-09-30",
		OrigemDataInicio:     store.OrigemColeta,
		OrigemDataExpiracao:  store.OrigemColeta,
		IndicacaoPromocional: false,
	}
	require.NoError(t, catalog.SaveMarca(ctx, store.Marca{ID: marcaID, Nome: "Marca X", NomeNorm: "marca x"}))
	require.NoError(t, catalog.SaveMercado(ctx, store.Mercado{ID: mercadoID, Nome: "Mercado 1"}))
	require.NoError(t, catalog.SaveOfertasForColeta(ctx, coletaID, []store.Oferta{oferta1}))

	// Create a product that needs sanitization but not backfill (already has correct size)
	prod2 := store.Produto{
		ID:         prodID2,
		Nome:       "Arroz Branco 5kg",
		NomeNorm:   store.NormalizarRotulo("arroz branco 5kg"),
		Categorias: []string{"graos"},
	}
	require.NoError(t, catalog.SaveProduto(ctx, prod2))

	// Create a product that is already clean
	prod3 := store.Produto{
		ID:         prodID3,
		Nome:       "Sabão em Pó",
		NomeNorm:   store.NormalizarRotulo("sabao em po"),
		Categorias: []string{"limpeza"},
	}
	require.NoError(t, catalog.SaveProduto(ctx, prod3))
	require.NoError(t, catalog.SaveDocumento(ctx, store.Documento{ID: docID, FonteID: "fonte1", MercadoID: mercadoID, Filename: "file1", Dia: "2026-09-01", Estado: store.EstadoConcluido}))
	require.NoError(t, catalog.SaveOferta(ctx, store.Oferta{ID: "oferta2", ProdutoID: prod3.ID, MercadoID: mercadoID, Valor: 10.0, Quantidades: []float64{1}, Medida: store.MedidaUnidade, DataInicio: "2026-09-01", DataExpiracao: "2026-09-30"}, []store.DocumentoID{docID}))

	// Run the merge service
	err = service.MergeProducts(ctx)
	require.NoError(t, err)

	// Verify prod1 is sanitized and backfilled
	updatedProd1, found, err := catalog.GetProduto(ctx, prodID1)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "creme de leite", updatedProd1.Nome)
	assert.Equal(t, "creme de leite", updatedProd1.NomeNorm)
	assert.Equal(t, "Creme de Leite 200g (Marca X)", updatedProd1.OriginalNome)

	updatedOferta1, found, err := catalog.GetOferta(ctx, oferta1.ID)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, []float64{200}, updatedOferta1.Quantidades)
	assert.Equal(t, store.MedidaG, updatedOferta1.Medida)

	// Verify prod2 is sanitized
	updatedProd2, found, err := catalog.GetProduto(ctx, prodID2)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "arroz branco", updatedProd2.Nome)
	assert.Equal(t, "arroz branco", updatedProd2.NomeNorm)
	assert.Equal(t, "Arroz Branco 5kg", updatedProd2.OriginalNome)

	// Verify prod3 is unchanged
	updatedProd3, found, err := catalog.GetProduto(ctx, prodID3)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "Sabão em Pó", updatedProd3.Nome)
	assert.Equal(t, "sabao em po", updatedProd3.NomeNorm)
	assert.Equal(t, "Sabão em Pó", updatedProd3.OriginalNome)
}
