package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/config"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/httpapi"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/infra/catalog"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/infra/coleta"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/infra/envio"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/infra/termo"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if cfg.DatabaseURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL required")
		os.Exit(1)
	}
	if cfg.GatewayToken == "" {
		fmt.Fprintln(os.Stderr, "GATEWAY_TOKEN required")
		os.Exit(1)
	}
	if cfg.ColetaURL == "" {
		fmt.Fprintln(os.Stderr, "COLETA_URL required")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	c, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer c.Close()

	deps := application.Deps{
		Cat:    catalog.Store{C: c},
		Coleta: coleta.Cliente{Base: cfg.ColetaURL, Token: cfg.GatewayToken},
		Envio:  envio.Gateway{Base: cfg.GatewayURL, Token: cfg.GatewayToken},
	}
	if cfg.GeminiAPIKey != "" {
		g := termo.Gemini{Key: cfg.GeminiAPIKey, Model: cfg.GeminiModel}
		deps.Termo = g
		deps.Intencao = g
		deps.Escolha = g
	}
	ag := application.New(deps)

	server := &http.Server{Addr: cfg.HTTPAddr, Handler: httpapi.New(ag, cfg.GatewayToken)}
	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()
	log.Printf("agente-ofertas listening on %s", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
