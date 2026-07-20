package extrator_test

import (
	"context"
	"encoding/json"
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

	// One Extract per page so a mid-run 429 still keeps earlier pages for review.
	var cands []domain.CandidatoOferta
	uso := &domain.UsoExtrator{Model: os.Getenv("GEMINI_MODEL")}
	var extractErr error
	for _, img := range images {
		pageCands, _, pageUso, err := g.Extract(ctx, []domain.PageImage{img})
		if err != nil {
			extractErr = err
			t.Logf("page=%d extract failed: %v", img.Page, err)
			break
		}
		cands = append(cands, pageCands...)
		if pageUso != nil {
			if uso.Model == "" {
				uso.Model = pageUso.Model
			}
			uso.PromptTokens += pageUso.PromptTokens
			uso.CacheTokens += pageUso.CacheTokens
			uso.OutputTokens += pageUso.OutputTokens
			uso.Paginas = append(uso.Paginas, pageUso.Paginas...)
		}
	}
	raw, err := json.Marshal(struct {
		Ofertas []domain.CandidatoOferta `json:"ofertas"`
	}{Ofertas: cands})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("ofertas=%d raw_bytes=%d pages_ok=%d", len(cands), len(raw), len(uso.Paginas))
	t.Logf("uso-extrator prompt=%d cache=%d output=%d",
		uso.PromptTokens, uso.CacheTokens, uso.OutputTokens)
	withPromo := 0
	for i, c := range cands {
		promo := ""
		if c.Promocao != nil {
			withPromo++
			b, _ := json.Marshal(c.Promocao)
			promo = " | " + string(b)
		}
		t.Logf("%02d. %s | %s | %.2f | %v%s%s", i+1, c.Marca, c.Produto, c.Valor, c.Quantidades, c.Medida, promo)
	}
	t.Logf("com promocao=%d / %d", withPromo, len(cands))

	// Side folder for human review (gitignored .data) — raw + uso, no Redis.
	reviewDir := filepath.Join(filepath.Dir(root), "live-page-by-page")
	if err := os.MkdirAll(reviewDir, 0o755); err == nil {
		_ = os.WriteFile(filepath.Join(reviewDir, "extrator-raw.json"), raw, 0o644)
		uso.ArtefatoPath = reviewDir
		if b, err := json.MarshalIndent(uso, "", "  "); err == nil {
			_ = os.WriteFile(filepath.Join(reviewDir, "uso-extrator.json"), b, 0o644)
		}
		imgLink := filepath.Join(reviewDir, "images")
		_ = os.Remove(imgLink)
		_ = os.Symlink(imgDir, imgLink)
	}

	if extractErr != nil {
		t.Fatalf("extract incomplete: %v (saved partial to %s)", extractErr, reviewDir)
	}
	// Fort Canoas 20–24/jul: ~95 priced offers (4-page grid + Girando Sol 800g inset).
	if len(cands) < 90 {
		t.Fatalf("recall too low: got %d ofertas (want ≥90)", len(cands))
	}
	if withPromo < 30 {
		t.Fatalf("promocao recall too low: got %d with promocao (want ≥30 on Fort)", withPromo)
	}
}
