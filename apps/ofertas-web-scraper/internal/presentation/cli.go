package presentation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/infra/vitrine"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

type Env struct {
	DatabaseURL string
	SeedPath    string
}

func LoadEnv() Env {
	return Env{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		SeedPath:    envOr("SEED_PATH", "./seed/mercados.json"),
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func openCatalog(ctx context.Context, env Env) (*store.Catalog, error) {
	return store.Open(ctx, env.DatabaseURL)
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
	catalog, err := openCatalog(ctx, env)
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

func RunColetar(ctx context.Context, env Env, produtoID string) error {
	catalog, err := openCatalog(ctx, env)
	if err != nil {
		return err
	}
	defer catalog.Close()
	return application.ColetarProduto(ctx, catalog, vitrine.Defaults(nil), store.ProdutoID(produtoID), hojeSaoPaulo())
}

func RunAll(ctx context.Context, env Env) error {
	catalog, err := openCatalog(ctx, env)
	if err != nil {
		return err
	}
	defer catalog.Close()
	produtos, err := catalog.ListProdutos(ctx)
	if err != nil {
		return err
	}
	dia := hojeSaoPaulo()
	vitrines := vitrine.Defaults(nil)
	for _, p := range produtos {
		log.Printf("coletar produto %s (%s)", p.ID, p.Nome)
		if err := application.ColetarProduto(ctx, catalog, vitrines, p.ID, dia); err != nil {
			return fmt.Errorf("produto %s: %w", p.ID, err)
		}
	}
	return nil
}
