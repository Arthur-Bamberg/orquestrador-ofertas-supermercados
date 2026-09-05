package shopfully_test

import (
	"os"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/shopfully"
)

func TestParseListagem_CollectsUniqueFlyerCards(t *testing.T) {
	html, err := os.ReadFile("testdata/listagem.html")
	if err != nil {
		t.Fatal(err)
	}

	got, err := shopfully.ParseListagem(string(html))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("got %d encartes: %+v", len(got), got)
	}

	want := []struct {
		id, mercado, path, vence string
	}{
		{"1379018", "Stok Center", "/canoas/supermercados/stok-center/promocoes/ultimas-ofertas-stok-center?flyerId=1379018&flyerPage=1", "Vence Quinta"},
		{"1379017", "Stok Center", "/canoas/supermercados/stok-center/promocoes/ultimas-ofertas-stok-center?flyerId=1379017&flyerPage=1", "Vence Segunda"},
		{"1376468", "Asun", "/canoas/supermercados/asun/promocoes/ultimas-ofertas-asun?flyerId=1376468&flyerPage=1", "Vence amanhã"},
		{"1376799", "Atacadão", "/canoas/supermercados/atacadao/promocoes/ultimas-ofertas-atacadao?flyerId=1376799&flyerPage=1", "Vence Segunda"},
	}
	for i, w := range want {
		e := got[i]
		if e.ID != w.id || e.Mercado != w.mercado || e.ViewerPath != w.path || e.Vencimento != w.vence {
			t.Fatalf("encarte[%d]=%+v want id=%s mercado=%s path=%s vence=%s", i, e, w.id, w.mercado, w.path, w.vence)
		}
	}
}
