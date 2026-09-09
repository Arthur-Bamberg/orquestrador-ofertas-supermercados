package domain_test

import (
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestEstadoAposColeta(t *testing.T) {
	cases := []struct {
		name               string
		ofertas, descartes int
		esgotada           bool
		want               store.EstadoDocumento
	}{
		{"esgotada com ofertas", 2, 0, true, store.EstadoConcluido},
		{"esgotada com descarte", 1, 1, true, store.EstadoParcial},
		{"esgotada vazia", 0, 0, true, store.EstadoConcluido},
		{"cortada com ofertas", 3, 0, false, store.EstadoParcial},
		{"cortada sem ofertas", 0, 0, false, store.EstadoFalhou},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := domain.EstadoAposColeta(tc.ofertas, tc.descartes, tc.esgotada); got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}
