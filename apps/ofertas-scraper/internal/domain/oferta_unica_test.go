package domain_test

import (
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

func TestChaveUnicaOferta_IgualSemDocumento(t *testing.T) {
	a := domain.Oferta{
		ID: "1", DocumentoID: "d1", ProdutoID: "p", MercadoID: "m",
		Valor: 10, Quantidades: []float64{500}, Medida: domain.MedidaG,
		DataInicio: "2026-07-18", DataExpiracao: "2026-07-19",
		OrigemDataInicio: domain.OrigemExtrator, OrigemDataExpiracao: domain.OrigemExtrator,
	}
	b := a
	b.ID = "2"
	b.DocumentoID = "d2"
	b.OrigemDataInicio = domain.OrigemFilename
	if domain.ChaveUnicaOferta(a) != domain.ChaveUnicaOferta(b) {
		t.Fatal("chave should ignore id, documento and origens")
	}
}

func TestChaveUnicaOferta_MarcaAusenteDistinta(t *testing.T) {
	base := domain.Oferta{
		ProdutoID: "p", MercadoID: "m", Valor: 1, Quantidades: []float64{1},
		Medida: domain.MedidaUnidade, DataInicio: "2026-07-01", DataExpiracao: "2026-07-02",
	}
	com := base
	id := domain.MarcaID("marca-1")
	com.MarcaID = &id
	if domain.ChaveUnicaOferta(base) == domain.ChaveUnicaOferta(com) {
		t.Fatal("absent marca must differ from present marca")
	}
}

func TestFingerprintPDF_Stable(t *testing.T) {
	a := domain.FingerprintPDF([]byte("pdf-bytes"))
	b := domain.FingerprintPDF([]byte("pdf-bytes"))
	if a != b || len(a) != 64 {
		t.Fatalf("got %q / %q", a, b)
	}
	if domain.FingerprintPDF([]byte("other")) == a {
		t.Fatal("different bytes should differ")
	}
}
