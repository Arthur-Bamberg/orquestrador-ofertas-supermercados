package extrator_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/extrator"
)

// Live smoke against a retained Fort Artefato (page-by-page Extrator).
// Run from apps/ofertas-scraper:
//
//	LIVE_EXTRATOR=1 go test ./internal/infra/extrator/ -run TestLiveFortArtefato -count=1 -timeout 10m
func TestLiveFortArtefato(t *testing.T) {
	if os.Getenv("LIVE_EXTRATOR") == "" {
		t.Skip("set LIVE_EXTRATOR=1 to run")
	}
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Fatal("GEMINI_API_KEY required")
	}

	root := os.Getenv("LIVE_ARTEFATO_DIR")
	if root == "" {
		root = filepath.Join(
			"..", "..", "..",
			".data", "artefatos", "fonte-fort", "2026-07-20",
			"RS_Fort_Semanal-4a-ED_Regional_20-a-24_JUL_26-Canoas-Final.pdf",
			"20260720T034602",
		)
	}
	imgDir := filepath.Join(root, "images")
	entries, err := os.ReadDir(imgDir)
	if err != nil {
		t.Fatalf("read images: %v", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".jpg" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatal("no page images")
	}

	var images []domain.PageImage
	for i, name := range names {
		b, err := os.ReadFile(filepath.Join(imgDir, name))
		if err != nil {
			t.Fatal(err)
		}
		images = append(images, domain.PageImage{Page: i + 1, JPEG: b})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	model := os.Getenv("GEMINI_MODEL")
	g, err := extrator.NewGemini(ctx, extrator.GeminiConfig{
		APIKey:             apiKey,
		Model:              model,
		PromptPath:         filepath.Join("..", "..", "..", "prompts", "extrator.txt"),
		ExtracaoSchemaPath: filepath.Join("..", "..", "..", "schemas", "extracao.json"),
		OfertaSchemaPath:   filepath.Join("..", "..", "..", "schemas", "oferta.json"),
	})
	if err != nil {
		t.Fatal(err)
	}

	cands, raw, err := g.Extract(ctx, images)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ofertas=%d raw_bytes=%d", len(cands), len(raw))
	for i, c := range cands {
		t.Logf("%02d. %s | %s | %.2f | %v%s", i+1, c.Marca, c.Produto, c.Valor, c.Quantidades, c.Medida)
	}
	// Fort Canoas 20–24/jul: ~95 priced offers (4-page grid + Girando Sol 800g inset).
	if len(cands) < 90 {
		t.Fatalf("recall too low: got %d ofertas (want ≥90)", len(cands))
	}
}
