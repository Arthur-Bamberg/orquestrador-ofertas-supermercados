package shopfully_test

import (
	"os"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/shopfully"
)

func TestParsePublicationID_ReadsDCFlyer(t *testing.T) {
	html, err := os.ReadFile("testdata/viewer.html")
	if err != nil {
		t.Fatal(err)
	}
	got, err := shopfully.ParsePublicationID(string(html))
	if err != nil {
		t.Fatal(err)
	}
	if got != "pt_br_858250" {
		t.Fatalf("got %q", got)
	}
}

func TestParsePublicationID_Missing(t *testing.T) {
	_, err := shopfully.ParsePublicationID(`<html><script>window.DCConfig = {}</script></html>`)
	if err == nil {
		t.Fatal("expected error")
	}
}
