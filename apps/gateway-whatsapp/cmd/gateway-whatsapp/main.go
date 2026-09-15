package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/google/uuid"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/config"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/httpapi"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/agente"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/canal"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/midia"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/pg"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/whatsapp"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/operador"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	store, err := pg.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer store.Close()

	var channel domain.Canal
	var graph *whatsapp.Cliente
	opt := httpapi.Options{VerifyToken: cfg.VerifyToken, AppSecret: cfg.AppSecret}
	if cfg.StubCanal {
		channel = &canal.Stub{}
		log.Print("canal stub (WHATSAPP_STUB=1)")
	} else {
		graph = &whatsapp.Cliente{
			Token:         cfg.WhatsAppToken,
			PhoneNumberID: cfg.PhoneNumberID,
			DisplayJID:    cfg.DisplayJID(),
			Version:       cfg.GraphVersion,
		}
		channel = graph
		opt.BaixarMidia = graph.BaixarMidia
		if graph.Pronto() {
			log.Printf("canal Cloud API phone_number_id=%s", cfg.PhoneNumberID)
		} else {
			log.Print("canal Cloud API não configurado (WHATSAPP_TOKEN / WHATSAPP_PHONE_NUMBER_ID)")
		}
	}

	deps := application.Deps{
		Allow:    domain.NovaAllowlist(cfg.Allowlist),
		Repo:     store,
		Canal:    channel,
		Midias:   midia.Local{Root: cfg.MidiaRoot},
		AckTexto: cfg.AckTexto,
		NewID:    func() string { return uuid.NewString() },
	}
	if cfg.AgenteURL != "" {
		deps.Agente = &agente.Cliente{URL: cfg.AgenteURL, Token: cfg.GatewayToken}
		log.Printf("agente em %s", cfg.AgenteURL)
	}
	gw := application.New(deps)

	ids, err := operador.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer ids.Close()

	server := &http.Server{Addr: cfg.HTTPAddr, Handler: httpapi.NewWith(gw, cfg.GatewayToken, cfg.CORSOrigin, ids, opt)}
	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()
	log.Printf("gateway-whatsapp listening on %s", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
