package domain_test

import (
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestOfertaDaColeta_VigenciaEOdiaEIndicacaoDoCartao(t *testing.T) {
	produto := store.Produto{ID: "p1", Nome: "Arroz integral", NomeNorm: "arroz integral"}
	cartao := domain.CartaoVitrine{
		Nome: "Arroz Integral Camil 1kg", Marca: "Camil", Valor: 8.9,
		Quantidades: []float64{1000}, Medida: store.MedidaG, IndicacaoPromocional: false,
	}
	got := domain.OfertaDaColeta("o1", produto, "mercado-asun", nil, "2026-09-05", cartao)
	if got.DataInicio != "2026-09-05" || got.DataExpiracao != "2026-09-05" {
		t.Fatalf("vigência=%s/%s", got.DataInicio, got.DataExpiracao)
	}
	if got.OrigemDataInicio != store.OrigemColeta || got.OrigemDataExpiracao != store.OrigemColeta {
		t.Fatalf("origens=%s/%s", got.OrigemDataInicio, got.OrigemDataExpiracao)
	}
	if got.IndicacaoPromocional {
		t.Fatal("site without badge must keep IndicacaoPromocional false")
	}
	if got.ProdutoID != "p1" || got.MercadoID != "mercado-asun" || got.Valor != 8.9 {
		t.Fatalf("got %#v", got)
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
	if domain.CartaoCompleto(domain.CartaoVitrine{Valor: 1, Medida: store.MedidaG}) {
		t.Fatal("missing quantidades")
	}
}
