package ops

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-api/internal/config"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

type Worker struct {
	catalog *store.Catalog
	cfg     config.Config
	log     *log.Logger
}

func NewWorker(catalog *store.Catalog, cfg config.Config, logger *log.Logger) *Worker {
	if logger == nil {
		logger = log.Default()
	}
	return &Worker{catalog: catalog, cfg: cfg, log: logger}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := w.runOnce(ctx); err != nil {
			w.log.Printf("ops worker: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) runOnce(ctx context.Context) error {
	op, ok, err := w.catalog.DequeueOperacao(ctx, time.Now())
	if err != nil || !ok {
		return err
	}
	w.log.Printf("op %s %s started", op.ID, op.Kind)
	output, execErr := w.execScraper(ctx, op)
	finished := time.Now().UTC()
	op.FinishedAt = &finished
	op.LogSnippet = tail(output, 8000)
	if execErr != nil {
		op.Status = store.OperacaoFailed
		op.Error = execErr.Error()
	} else {
		op.Status = store.OperacaoSucceeded
		op.Error = ""
	}
	if err := w.catalog.SaveOperacao(ctx, op); err != nil {
		return err
	}
	w.log.Printf("op %s %s finished status=%s", op.ID, op.Kind, op.Status)
	return nil
}

func (w *Worker) execScraper(ctx context.Context, op store.OperacaoPipeline) (string, error) {
	args := []string{"run", "./cmd/ofertas-scraper"}
	switch op.Kind {
	case store.OperacaoRun:
		args = append(args, "run")
	case store.OperacaoDiscover:
		if op.FonteID == "" {
			return "", fmt.Errorf("fonteId obrigatório")
		}
		args = append(args, "discover", string(op.FonteID))
	case store.OperacaoReprocess:
		if op.DocumentoID == "" {
			return "", fmt.Errorf("documentoId obrigatório")
		}
		args = append(args, "reprocess", string(op.DocumentoID))
	default:
		return "", fmt.Errorf("op kind desconhecido: %s", op.Kind)
	}
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = w.cfg.ScraperDir
	cmd.Env = append(os.Environ(),
		"UPSTASH_REDIS_REST_URL="+w.cfg.UpstashURL,
		"UPSTASH_REDIS_REST_TOKEN="+w.cfg.UpstashToken,
		"ARTEFATO_ROOT="+w.cfg.ArtefatoRoot,
	)
	raw, err := cmd.CombinedOutput()
	return string(raw), err
}

func tail(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return strings.TrimLeft(s[len(s)-max:], "\n")
}
