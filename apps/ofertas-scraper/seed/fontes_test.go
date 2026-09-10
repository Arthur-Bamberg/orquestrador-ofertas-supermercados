package seed_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

func TestFontesSeed_FortStokBourbonSaoSiteViaEEncarte(t *testing.T) {
	raw, err := os.ReadFile("fontes.json")
	if err != nil {
		t.Fatal(err)
	}
	var seed struct {
		Fontes []domain.Fonte `json:"fontes"`
	}
	if err := json.Unmarshal(raw, &seed); err != nil {
		t.Fatal(err)
	}
	got := map[domain.FonteID]domain.TipoFonte{}
	urls := map[domain.FonteID]string{}
	filtros := map[domain.FonteID]string{}
	for _, f := range seed.Fontes {
		got[f.ID] = f.TipoOuEncarte()
		urls[f.ID] = f.URL
		filtros[f.ID] = f.FiltroNomeDocumento
	}
	want := map[domain.FonteID]domain.TipoFonte{
		"fonte-fort":      domain.TipoSite,
		"fonte-stok":      domain.TipoSite,
		"fonte-bourbon":   domain.TipoSite,
		"fonte-carrefour": domain.TipoSite,
		"fonte-via":       domain.TipoEncarte,
	}
	if len(got) != len(want) {
		t.Fatalf("fontes=%v", got)
	}
	for id, tipo := range want {
		if got[id] != tipo {
			t.Fatalf("%s tipo=%q want %q", id, got[id], tipo)
		}
	}
	if urls["fonte-fort"] != "https://www.fortatacadista.com.br/" {
		t.Fatalf("fonte-fort url=%q", urls["fonte-fort"])
	}
	if urls["fonte-stok"] != "https://www.stokonline.com.br/" {
		t.Fatalf("fonte-stok url=%q", urls["fonte-stok"])
	}
	if urls["fonte-bourbon"] != "https://www.zaffari.com.br/" {
		t.Fatalf("fonte-bourbon url=%q", urls["fonte-bourbon"])
	}
	if urls["fonte-carrefour"] != "https://mercado.carrefour.com.br/" {
		t.Fatalf("fonte-carrefour url=%q", urls["fonte-carrefour"])
	}
	if filtros["fonte-fort"] != "" {
		t.Fatalf("fonte-fort filtroNomeDocumento=%q", filtros["fonte-fort"])
	}
}
