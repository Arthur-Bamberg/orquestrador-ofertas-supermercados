package presentation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrine"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/presentation/httpapi"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

type Env struct {
	DatabaseURL string
	SeedPath    string
	HTTPAddr    string
	Token       string
}

func LoadEnv() Env {
	return Env{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		SeedPath:    envOr("SEED_PATH", "./seed/mercados.json"),
		HTTPAddr:    envOr("HTTP_ADDR", ":8092"),
		Token:       os.Getenv("GATEWAY_TOKEN"),
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func hojeSaoPaulo() string {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc).Format("2006-01-02")
}

type seedFile struct {
	Mercados []store.Mercado `json:"mercados"`
}

func RunSeed(ctx context.Context, env Env) error {
	catalog, err := store.Open(ctx, env.DatabaseURL)
	if err != nil {
		return err
	}
	defer catalog.Close()
	raw, err := os.ReadFile(env.SeedPath)
	if err != nil {
		return err
	}
	var seed seedFile
	if err := json.Unmarshal(raw, &seed); err != nil {
		return err
	}
	repo := store.NewMercadoRepo(catalog)
	for _, m := range seed.Mercados {
		if err := repo.Save(ctx, m); err != nil {
			return err
		}
		log.Printf("seed mercado %s (%s)", m.ID, m.Nome)
	}
	return nil
}

func RunColetar(ctx context.Context, env Env, termo string) error {
	catalog, err := store.Open(ctx, env.DatabaseURL)
	if err != nil {
		return err
	}
	defer catalog.Close()
	ofertas, err := application.Coletar(ctx, catalog, vitrine.Defaults(nil), termo, hojeSaoPaulo())
	if err != nil {
		return err
	}
	log.Printf("coleta %q: %d ofertas", termo, len(ofertas))
	return nil
}

func RunServe(env Env) error {
	if env.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL obrigatório")
	}
	if env.Token == "" {
		return fmt.Errorf("GATEWAY_TOKEN obrigatório")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	catalog, err := store.Open(ctx, env.DatabaseURL)
	if err != nil {
		return err
	}
	defer catalog.Close()
	server := &http.Server{
		Addr: env.HTTPAddr,
		Handler: httpapi.New(httpapi.Deps{
			Catalog:  catalog,
			Vitrines: vitrine.Defaults(nil),
			Hoje:     hojeSaoPaulo,
			Token:    env.Token,
		}),
	}
	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()
	log.Printf("ofertas-scraper-v2 listening on %s", env.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
