package application

import (
	"context"
	"log"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
)

// Shopfully is the HTTP port for listing, page URLs and JPEG download.
type Shopfully interface {
	Listagem(ctx context.Context, path string) ([]domain.Encarte, error)
	Paginas(ctx context.Context, encarte domain.Encarte) ([]domain.Pagina, error)
	Download(ctx context.Context, rawURL string) ([]byte, error)
}

// ImagemStore persists a page JPEG of an Encarte.
type ImagemStore interface {
	Save(mercado, encarteID string, pagina domain.Pagina, body []byte) (relPath string, err error)
}

// Job discovers Shopfully flyers and downloads page images of Mercados we already have.
type Job struct {
	ListagemPath string
	Mercados     []string
	Client       Shopfully
	Store        ImagemStore
}

// Result is the observable outcome of one run.
type Result struct {
	Listados  int
	Filtrados int
	Baixados  int
	Falhas    int
	Arquivos  []string
}

func (j Job) Run(ctx context.Context) (Result, error) {
	encartes, err := j.Client.Listagem(ctx, j.ListagemPath)
	if err != nil {
		return Result{}, err
	}
	filtrados := domain.FiltrarEncartes(encartes, j.Mercados)
	out := Result{Listados: len(encartes), Filtrados: len(filtrados)}
	for _, e := range filtrados {
		paginas, err := j.Client.Paginas(ctx, e)
		if err != nil {
			log.Printf("skip encarte %s (%s): %v", e.ID, e.Mercado, err)
			out.Falhas++
			continue
		}
		for _, p := range paginas {
			body, err := j.Client.Download(ctx, p.URL)
			if err != nil {
				log.Printf("skip pagina %s p%d: %v", e.ID, p.Numero, err)
				out.Falhas++
				continue
			}
			rel, err := j.Store.Save(e.Mercado, e.ID, p, body)
			if err != nil {
				return out, err
			}
			out.Arquivos = append(out.Arquivos, rel)
			out.Baixados++
		}
	}
	return out, nil
}
