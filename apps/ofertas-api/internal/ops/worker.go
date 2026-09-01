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

func scraperCLIArgs(op store.OperacaoPipeline) ([]string, error) {
	switch op.Kind {
	case store.OperacaoRun:
		return []string{"run"}, nil
	case store.OperacaoDiscover:
		if op.FonteID == "" {
			return nil, fmt.Errorf("fonteId obrigatório")
		}
		return []string{"discover", string(op.FonteID)}, nil
	case store.OperacaoReprocess:
		if op.DocumentoID == "" {
			return nil, fmt.Errorf("documentoId obrigatório")
		}
		return []string{"reprocess", string(op.DocumentoID)}, nil
	default:
		return nil, fmt.Errorf("op kind desconhecido: %s", op.Kind)
	}
}

func scraperExec(cfg config.Config, op store.OperacaoPipeline) (name, dir string, args []string, err error) {
	cli, err := scraperCLIArgs(op)
	if err != nil {
		return "", "", nil, err
	}
	if cfg.ScraperBin != "" {
		return cfg.ScraperBin, cfg.ScraperDir, cli, nil
	}
	return "go", cfg.ScraperDir, append([]string{"run", "./cmd/ofertas-scraper"}, cli...), nil
}

type scraperCommandFn func(ctx context.Context, name, dir string, env []string, args ...string) (string, error)

var runScraperCommand scraperCommandFn = defaultRunScraperCommand

func defaultRunScraperCommand(ctx context.Context, name, dir string, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = env
	raw, err := cmd.CombinedOutput()
	return string(raw), err
}

func (w *Worker) execScraper(ctx context.Context, op store.OperacaoPipeline) (string, error) {
	name, dir, args, err := scraperExec(w.cfg, op)
	if err != nil {
		return "", err
	}
	env := append(os.Environ(),
		"DATABASE_URL="+w.cfg.DatabaseURL,
		"ARTEFATO_ROOT="+w.cfg.ArtefatoRoot,
	)
	return runScraperCommand(ctx, name, dir, env, args...)
}

func tail(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return strings.TrimLeft(s[len(s)-max:], "\n")
}
