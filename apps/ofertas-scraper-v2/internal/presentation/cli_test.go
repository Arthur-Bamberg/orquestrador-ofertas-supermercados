package presentation

import (
	"path/filepath"
	"testing"
)

func TestLoadMercados_ReadsCatalogNamesWeHave(t *testing.T) {
	path := filepath.Join("..", "..", "seed", "mercados.json")
	got, err := LoadMercados(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Fort Atacadista", "Stok Center", "Via Atacadista"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v", got)
		}
	}
}
