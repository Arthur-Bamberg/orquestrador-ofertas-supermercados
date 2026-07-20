package extrator_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/extrator"
)

// Page-1 only against a high-res JPEG (probe insets / small packs).
//
//	LIVE_EXTRATOR=1 LIVE_PAGE1_JPEG=/tmp/fort-hires/200/page-1.jpg \
//	  go test ./internal/infra/extrator/ -run TestLiveFortPage1Hires -count=1 -timeout 5m -v
func TestLiveFortPage1Hires(t *testing.T) {
	if os.Getenv("LIVE_EXTRATOR") == "" {
		t.Skip("set LIVE_EXTRATOR=1 to run")
	}
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Fatal("GEMINI_API_KEY required")
	}
	jpegPath := os.Getenv("LIVE_PAGE1_JPEG")
	if jpegPath == "" {
		t.Skip("set LIVE_PAGE1_JPEG to a page-1 JPEG path")
	}
	b, err := os.ReadFile(jpegPath)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	g, err := extrator.NewGemini(ctx, extrator.GeminiConfig{
		APIKey:             apiKey,
		Model:              os.Getenv("GEMINI_MODEL"),
		PromptPath:         filepath.Join("..", "..", "..", "prompts", "extrator.txt"),
		ExtracaoSchemaPath: filepath.Join("..", "..", "..", "schemas", "extracao.json"),
		OfertaSchemaPath:   filepath.Join("..", "..", "..", "schemas", "oferta.json"),
	})
	if err != nil {
		t.Fatal(err)
	}

	cands, _, err := g.Extract(ctx, []domain.PageImage{{Page: 1, JPEG: b}})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("page1 ofertas=%d", len(cands))
	hasGirando4k := false
	hasGirando800 := false
	for i, c := range cands {
		t.Logf("%02d. %s | %s | %.2f | %v%s", i+1, c.Marca, c.Produto, c.Valor, c.Quantidades, c.Medida)
		if c.Valor == 24.9 || c.Valor == 24.90 {
			hasGirando4k = true
		}
		if c.Valor == 4.98 || c.Valor == 4.9 {
			hasGirando800 = true
		}
	}
	if !hasGirando4k {
		t.Error("missing Girando Sol 4kg (~24.90)")
	}
	if !hasGirando800 {
		t.Error("missing Girando Sol 800g inset (~4.98)")
	}
}
