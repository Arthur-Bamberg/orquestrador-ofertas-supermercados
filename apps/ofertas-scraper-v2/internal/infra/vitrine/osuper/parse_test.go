package osuper_test

import (
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrine/osuper"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestMedidaDoCartao(t *testing.T) {
	cases := []struct {
		unit string
		frac float64
		m    store.Medida
		q    float64
	}{
		{"UN", 1, store.MedidaUnidade, 1},
		{"KG", 0.13, store.MedidaG, 1000},
		{"G", 500, store.MedidaG, 500},
		{"L", 1, store.MedidaML, 1000},
		{"ML", 330, store.MedidaML, 330},
		{"CX", 1, store.MedidaUnidade, 1},
		{"", 2, store.MedidaUnidade, 2},
	}
	for _, tc := range cases {
		m, q := osuper.MedidaDoCartao(tc.unit, tc.frac)
		if m != tc.m || q != tc.q {
			t.Fatalf("%q frac=%v → %s %v want %s %v", tc.unit, tc.frac, m, q, tc.m, tc.q)
		}
	}
}
