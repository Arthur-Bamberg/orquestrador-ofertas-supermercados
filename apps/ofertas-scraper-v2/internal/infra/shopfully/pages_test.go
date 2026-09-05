package shopfully_test

import (
	"os"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/shopfully"
)

func TestBestPageImages_SinglePagePublication(t *testing.T) {
	raw, err := os.ReadFile("testdata/pages_stok.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := shopfully.BestPageImages(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d páginas: %+v", len(got), got)
	}
	if got[0].Numero != 1 {
		t.Fatalf("numero=%d", got[0].Numero)
	}
	want := "https://pt-br-media-publications.shopfully.cloud/publications/page_assets/858250/1/page_1_level_5_1147866633.jpeg"
	if got[0].URL != want {
		t.Fatalf("url=%s", got[0].URL)
	}
}

func TestBestPageImages_UsesHighestResolution(t *testing.T) {
	raw, err := os.ReadFile("testdata/pages_atacadao_1.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := shopfully.BestPageImages(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 10 {
		t.Fatalf("got %d páginas (bundle 1 tem 10)", len(got))
	}
	if got[0].Numero != 1 || got[9].Numero != 10 {
		t.Fatalf("ordem %+v", got)
	}
	want := "https://pt-br-media-publications.shopfully.cloud/publications/page_assets/856415/1/page_1_level_5_1233176198.jpeg"
	if got[0].URL != want {
		t.Fatalf("page1 url=%s", got[0].URL)
	}
}

func TestBundlePaths_ListsAllParts(t *testing.T) {
	raw, err := os.ReadFile("testdata/pages_atacadao_1.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := shopfully.BundlePaths(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	if got[0] != "https://shopfully-publication-api.global.ssl.fastly.net/publication_pages/pt_br/856415/1" {
		t.Fatalf("first=%s", got[0])
	}
	if got[1] != "https://shopfully-publication-api.global.ssl.fastly.net/publication_pages/pt_br/856415/2" {
		t.Fatalf("second=%s", got[1])
	}
}

func TestPublicationBundlePath(t *testing.T) {
	raw, err := os.ReadFile("testdata/publication_stok.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := shopfully.PublicationBundlePath(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://shopfully-publication-api.global.ssl.fastly.net/publication_pages/pt_br/858250/1" {
		t.Fatalf("got %s", got)
	}
}
