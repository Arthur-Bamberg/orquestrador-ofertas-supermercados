package extrator_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/extrator"
)

func TestStub_ReturnsCandidatosAndRaw(t *testing.T) {
	s := extrator.Stub{Candidatos: []domain.CandidatoOferta{{
		Produto: "Arroz", Valor: 1, Quantidades: []float64{1}, Medida: "g",
		DataInicio: "2026-07-18", DataExpiracao: "2026-07-20",
	}}}
	cands, raw, uso, err := s.Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 1 || cands[0].Produto != "Arroz" {
		t.Fatalf("candidatos=%v", cands)
	}
	if len(raw) == 0 {
		t.Fatal("raw empty")
	}
	if uso == nil || uso.Provider != domain.ExtratorProviderStub {
		t.Fatalf("stub uso=%#v", uso)
	}
}

func TestStub_Unavailable(t *testing.T) {
	_, _, _, err := extrator.Unavailable().Extract(context.Background(), nil)
	if !errors.Is(err, domain.ErrExtratorIndisponivel) {
		t.Fatalf("err=%v", err)
	}
	if !extrator.IsIndisponivel(err) {
		t.Fatal("IsIndisponivel false")
	}
}
