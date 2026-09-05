package store_test

import (
	"context"
	"testing"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store/storetest"
)

func TestNormalizarRotulo_colapsaEspacosELowercase(t *testing.T) {
	got := store.NormalizarRotulo("  Arroz   Integral ")
	if got != "arroz integral" {
		t.Fatalf("got %q", got)
	}
}

func TestMatchOrCreateMarca_ReusaPorNomeNorm(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	repo := store.NewMarcaRepo(catalog)
	first, err := store.MatchOrCreateMarca(ctx, repo, "  Camil  ", "marca-1")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != "marca-1" || first.NomeNorm != "camil" || first.Nome != "  Camil  " {
		t.Fatalf("created %#v", first)
	}
	second, err := store.MatchOrCreateMarca(ctx, repo, "CAMIL", "marca-2")
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != "marca-1" {
		t.Fatalf("reuse want marca-1, got %#v", second)
	}
}
