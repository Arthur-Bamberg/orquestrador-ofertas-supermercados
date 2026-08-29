package ops

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-api/internal/config"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store/storetest"
)

func TestMain(m *testing.M) {
	code := m.Run()
	storetest.Stop()
	os.Exit(code)
}

func TestScraperArgs(t *testing.T) {
	t.Run("run", func(t *testing.T) {
		got, err := scraperArgs(store.OperacaoPipeline{Kind: store.OperacaoRun})
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"run", "./cmd/ofertas-scraper", "run"}
		if len(got) != len(want) || got[0] != want[0] || got[2] != want[2] {
			t.Fatalf("got=%v want=%v", got, want)
		}
	})
	t.Run("discover_exige_fonteId", func(t *testing.T) {
		if _, err := scraperArgs(store.OperacaoPipeline{Kind: store.OperacaoDiscover}); err == nil {
			t.Fatal("expected error")
		}
		got, err := scraperArgs(store.OperacaoPipeline{Kind: store.OperacaoDiscover, FonteID: "f1"})
		if err != nil {
			t.Fatal(err)
		}
		if got[len(got)-1] != "f1" {
			t.Fatalf("got=%v", got)
		}
	})
	t.Run("reprocess_exige_documentoId", func(t *testing.T) {
		if _, err := scraperArgs(store.OperacaoPipeline{Kind: store.OperacaoReprocess}); err == nil {
			t.Fatal("expected error")
		}
		got, err := scraperArgs(store.OperacaoPipeline{Kind: store.OperacaoReprocess, DocumentoID: "d1"})
		if err != nil {
			t.Fatal(err)
		}
		if got[len(got)-1] != "d1" {
			t.Fatalf("got=%v", got)
		}
	})
}

func TestRunOnce_PersistsSucceededAndFailed(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	w := NewWorker(catalog, config.Config{ScraperDir: "/tmp/scraper", ArtefatoRoot: "/tmp/art"}, log.New(&bytes.Buffer{}, "", 0))

	t.Cleanup(func() { runGoCommand = defaultRunGoCommand })

	t.Run("succeeded", func(t *testing.T) {
		if err := catalog.EnqueueOperacao(ctx, store.OperacaoPipeline{ID: "op-ok", Kind: store.OperacaoRun}); err != nil {
			t.Fatal(err)
		}
		runGoCommand = func(_ context.Context, dir string, _ []string, args ...string) (string, error) {
			if dir != "/tmp/scraper" {
				t.Fatalf("dir=%s", dir)
			}
			if len(args) < 3 || args[2] != "run" {
				t.Fatalf("args=%v", args)
			}
			return "job ok\n", nil
		}
		if err := w.runOnce(ctx); err != nil {
			t.Fatal(err)
		}
		got, ok, err := catalog.GetOperacao(ctx, "op-ok")
		if err != nil || !ok || got.Status != store.OperacaoSucceeded {
			t.Fatalf("ok=%v err=%v got=%+v", ok, err, got)
		}
		if got.LogSnippet != "job ok\n" {
			t.Fatalf("log=%q", got.LogSnippet)
		}
	})

	t.Run("failed", func(t *testing.T) {
		if err := catalog.EnqueueOperacao(ctx, store.OperacaoPipeline{ID: "op-fail", Kind: store.OperacaoRun}); err != nil {
			t.Fatal(err)
		}
		runGoCommand = func(context.Context, string, []string, ...string) (string, error) {
			return "boom\n", errors.New("exit 1")
		}
		if err := w.runOnce(ctx); err != nil {
			t.Fatal(err)
		}
		got, ok, err := catalog.GetOperacao(ctx, "op-fail")
		if err != nil || !ok || got.Status != store.OperacaoFailed {
			t.Fatalf("ok=%v err=%v got=%+v", ok, err, got)
		}
		if got.Error == "" {
			t.Fatal("expected error recorded")
		}
	})

	t.Run("log_longo_grava_cauda", func(t *testing.T) {
		if err := catalog.EnqueueOperacao(ctx, store.OperacaoPipeline{ID: "op-log", Kind: store.OperacaoRun}); err != nil {
			t.Fatal(err)
		}
		long := strings.Repeat("x", 9000)
		runGoCommand = func(context.Context, string, []string, ...string) (string, error) {
			return long, nil
		}
		if err := w.runOnce(ctx); err != nil {
			t.Fatal(err)
		}
		got, ok, err := catalog.GetOperacao(ctx, "op-log")
		if err != nil || !ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
		if len(got.LogSnippet) != 8000 {
			t.Fatalf("len=%d want 8000", len(got.LogSnippet))
		}
		if !strings.HasSuffix(got.LogSnippet, "xxxx") {
			t.Fatalf("snippet=%q", got.LogSnippet[len(got.LogSnippet)-8:])
		}
	})
}
