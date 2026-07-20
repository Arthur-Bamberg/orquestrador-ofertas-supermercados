package domain_test

import (
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

func TestNormalizarRotulo_colapsaEspacosELowercase(t *testing.T) {
	got := domain.NormalizarRotulo("  Arroz   Integral ")
	if got != "arroz integral" {
		t.Fatalf("got %q", got)
	}
}

func TestUnirCategorias_uneSemDuplicar(t *testing.T) {
	got := domain.UnirCategorias(
		[]string{"mercearia", "arroz"},
		[]string{"Arroz", "grãos"},
	)
	want := []string{"mercearia", "arroz", "grãos"}
	if len(got) != len(want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v want %#v", got, want)
		}
	}
}

func TestEstadoAposValidacao(t *testing.T) {
	cases := []struct {
		ofertas, falhas int
		want            domain.EstadoDocumento
	}{
		{0, 0, domain.EstadoFalhou},
		{0, 3, domain.EstadoFalhou},
		{2, 0, domain.EstadoConcluido},
		{2, 1, domain.EstadoParcial},
	}
	for _, tc := range cases {
		got := domain.EstadoAposValidacao(tc.ofertas, tc.falhas)
		if got != tc.want {
			t.Fatalf("ofertas=%d falhas=%d: got %q want %q", tc.ofertas, tc.falhas, got, tc.want)
		}
	}
}

func TestDeveReprocessar(t *testing.T) {
	if domain.DeveReprocessar(domain.EstadoConcluido) {
		t.Fatal("concluido não reprocessa")
	}
	if domain.DeveReprocessar(domain.EstadoParcial) {
		t.Fatal("parcial não reprocessa")
	}
	if !domain.DeveReprocessar(domain.EstadoFalhou) {
		t.Fatal("falhou deve reprocessar")
	}
	if !domain.DeveReprocessar(domain.EstadoProcessando) {
		t.Fatal("processando órfão deve reprocessar")
	}
	// Legacy Redis value — unknown states hit default (no special-case).
	if !domain.DeveReprocessar(domain.EstadoDocumento("descoberto")) {
		t.Fatal("estado legado desconhecido deve reprocessar")
	}
}
