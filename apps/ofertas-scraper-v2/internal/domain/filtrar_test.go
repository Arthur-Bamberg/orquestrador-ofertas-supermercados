package domain

import "testing"

func TestFiltrarEncartes_KeepsOnlyMercadosWeHave(t *testing.T) {
	encartes := []Encarte{
		{ID: "1", Mercado: "Stok Center"},
		{ID: "2", Mercado: "Asun"},
		{ID: "3", Mercado: "stok  center"},
		{ID: "4", Mercado: "Fort Atacadista"},
		{ID: "5", Mercado: "Via Atacadista"},
		{ID: "6", Mercado: "Atacadão"},
	}
	got := FiltrarEncartes(encartes, []string{"Fort Atacadista", "Stok Center", "Via Atacadista"})
	if len(got) != 4 {
		t.Fatalf("got %d: %+v", len(got), got)
	}
	wantIDs := []string{"1", "3", "4", "5"}
	for i, id := range wantIDs {
		if got[i].ID != id {
			t.Fatalf("got[%d]=%+v want id=%s", i, got[i], id)
		}
	}
}

func TestFiltrarEncartes_EmptyMercadosKeepsNone(t *testing.T) {
	got := FiltrarEncartes([]Encarte{{ID: "1", Mercado: "Stok Center"}}, nil)
	if len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}
