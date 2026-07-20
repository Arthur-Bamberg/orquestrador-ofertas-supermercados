package extrator_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

// Optional hi-res page-1 probe (insets / small packs).
//
//	LIVE_EXTRATOR=1 LIVE_PAGE1_JPEG=/path/to/page-1.jpg \
//	  go test ./internal/infra/extrator/ -run TestLiveFortPage1Hires -count=1 -timeout 5m -v
func TestLiveFortPage1Hires(t *testing.T) {
	if os.Getenv("LIVE_EXTRATOR") == "" {
		t.Skip("set LIVE_EXTRATOR=1 to run")
	}
	path := os.Getenv("LIVE_PAGE1_JPEG")
	if path == "" {
		t.Skip("set LIVE_PAGE1_JPEG to a JPEG path")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ext := newLiveExtrator(t, ctx)
	cands, _, _, err := ext.Extract(ctx, []domain.PageImage{{Page: 1, JPEG: b}})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("page1 hires ofertas=%d", len(cands))
	for i, c := range cands {
		t.Logf("%02d. %s | %s | %.2f | %v%s", i+1, c.Marca, c.Produto, c.Valor, c.Quantidades, c.Medida)
	}
}
