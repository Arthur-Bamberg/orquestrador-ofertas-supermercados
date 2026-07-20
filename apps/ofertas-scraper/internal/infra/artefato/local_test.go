package artefato_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/artefato"
)

func TestLocalStore_SavePDFUnderAttemptPath(t *testing.T) {
	root := t.TempDir()
	store := artefato.NewLocalStore(root)
	doc := domain.Documento{
		FonteID:  "fonte1",
		Filename: "encarte.pdf",
		Dia:      "2026-07-18",
	}
	err := store.SavePDF(context.Background(), doc, "20260718T120000", []byte("%PDF"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "fonte1", "2026-07-18", "encarte.pdf", "20260718T120000", "original.pdf")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "%PDF" {
		t.Fatalf("content=%q", b)
	}
}
