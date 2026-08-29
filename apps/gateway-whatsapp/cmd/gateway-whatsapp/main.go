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
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/canal"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/midia"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/pg"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/whatsapp"
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
	var wa *whatsapp.Cliente
	if cfg.StubCanal {
		channel = &canal.Stub{}
		log.Print("canal stub (WHATSAPP_STUB=1)")
	} else {
		wa, err = whatsapp.Ligar(ctx, cfg.SessionPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		channel = wa
		defer wa.Disconnect()
	}

	gw := application.New(application.Deps{
		Allow:    domain.NovaAllowlist(cfg.Allowlist),
		Repo:     store,
		Canal:    channel,
		Midias:   midia.Local{Root: cfg.MidiaRoot},
		AckTexto: cfg.AckTexto,
		NewID:    func() string { return uuid.NewString() },
	})

	if wa != nil {
		wa.SetHandler(func(_ context.Context, in application.Entrada) {
			go func() {
				if _, err := gw.Receber(context.Background(), in); err != nil {
					log.Printf("receber: %v", err)
				}
			}()
		})
		if err := wa.Connect(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	server := &http.Server{Addr: cfg.HTTPAddr, Handler: httpapi.New(gw, cfg.GatewayToken)}
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
