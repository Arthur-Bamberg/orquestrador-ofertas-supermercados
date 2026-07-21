package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-api/internal/api"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-api/internal/config"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-api/internal/ops"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if cfg.UpstashURL == "" || cfg.UpstashToken == "" {
		fmt.Fprintln(os.Stderr, "UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN required")
		os.Exit(1)
	}

	client := store.NewClient(cfg.UpstashURL, cfg.UpstashToken, nil)
	catalog := store.NewCatalog(client)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go ops.NewWorker(catalog, cfg, log.Default()).Run(ctx)

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: api.New(catalog, cfg),
	}
	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()

	log.Printf("ofertas-api listening on %s", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
