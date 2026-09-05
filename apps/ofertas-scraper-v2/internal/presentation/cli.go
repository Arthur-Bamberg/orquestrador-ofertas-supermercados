package presentation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/artefato"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/shopfully"
)

type Env struct {
	ListagemPath string
	ArtefatoRoot string
	SeedPath     string
}

func LoadEnv() Env {
	return Env{
		ListagemPath: envOr("SHOPFULLY_LISTAGEM_PATH", "/canoas/supermercados"),
		ArtefatoRoot: envOr("ARTEFATO_ROOT", "./.data/artefatos"),
		SeedPath:     envOr("SEED_PATH", "./seed/mercados.json"),
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type seedFile struct {
	Mercados []struct {
		Nome string `json:"nome"`
	} `json:"mercados"`
}

func LoadMercados(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var seed seedFile
	if err := json.Unmarshal(raw, &seed); err != nil {
		return nil, err
	}
	var nomes []string
	for _, m := range seed.Mercados {
		if m.Nome != "" {
			nomes = append(nomes, m.Nome)
		}
	}
	if len(nomes) == 0 {
		return nil, fmt.Errorf("seed sem mercados: %s", path)
	}
	return nomes, nil
}

func newJob(env Env) (application.Job, error) {
	nomes, err := LoadMercados(env.SeedPath)
	if err != nil {
		return application.Job{}, err
	}
	httpClient := &http.Client{Timeout: 90 * time.Second}
	return application.Job{
		ListagemPath: env.ListagemPath,
		Mercados:     nomes,
		Client:       shopfully.New(httpClient, shopfully.Config{}),
		Store:        artefato.New(env.ArtefatoRoot),
	}, nil
}

func RunDiscover(ctx context.Context, env Env) error {
	job, err := newJob(env)
	if err != nil {
		return err
	}
	encartes, err := job.Client.Listagem(ctx, job.ListagemPath)
	if err != nil {
		return err
	}
	filtrados := domain.FiltrarEncartes(encartes, job.Mercados)
	fmt.Printf("listados=%d filtrados=%d mercados=%v\n", len(encartes), len(filtrados), job.Mercados)
	for _, e := range filtrados {
		fmt.Printf("%s\t%s\t%s\t%s\n", e.ID, e.Mercado, e.Vencimento, e.ViewerPath)
	}
	return nil
}

func RunDownload(ctx context.Context, env Env) error {
	job, err := newJob(env)
	if err != nil {
		return err
	}
	got, err := job.Run(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("listados=%d filtrados=%d baixados=%d falhas=%d root=%s\n",
		got.Listados, got.Filtrados, got.Baixados, got.Falhas, env.ArtefatoRoot)
	for _, p := range got.Arquivos {
		fmt.Println(p)
	}
	return nil
}
