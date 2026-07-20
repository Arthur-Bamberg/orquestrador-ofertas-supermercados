package filenamedate_test

import (
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/filenamedate"
)

func TestParser_extraiDataISOComoFim(t *testing.T) {
	p := filenamedate.Parser{}
	got := p.Parse("ofertas_atacado_20260725.pdf")
	if got.DataExpiracao != "2026-07-25" || got.DataInicio != "" {
		t.Fatalf("got %#v", got)
	}
	got = p.Parse("flyer-2026-07-18-final.pdf")
	if got.DataExpiracao != "2026-07-18" || got.DataInicio != "" {
		t.Fatalf("got %#v", got)
	}
	got = p.Parse("sem-data.pdf")
	if got.DataInicio != "" || got.DataExpiracao != "" {
		t.Fatalf("got %#v", got)
	}
}

func TestParser_extraiIntervaloFort(t *testing.T) {
	p := filenamedate.Parser{}
	got := p.Parse("MS_Fort_FDS_18-e-19_JUL_26.pdf")
	if got.DataInicio != "2026-07-18" || got.DataExpiracao != "2026-07-19" {
		t.Fatalf("got %#v", got)
	}
	got = p.Parse("DF_Fort_Fim_De_Semana_18-E-19_JUL_26-Final.pdf")
	if got.DataInicio != "2026-07-18" || got.DataExpiracao != "2026-07-19" {
		t.Fatalf("got %#v", got)
	}
	got = p.Parse("RS_Fort_FDS_Regional_18-e-19_JUL_26-Canoas-Final.pdf")
	if got.DataInicio != "2026-07-18" || got.DataExpiracao != "2026-07-19" {
		t.Fatalf("canoas: %#v", got)
	}
}
