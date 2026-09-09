package vitrinehttp

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
)

type Client struct {
	HTTP    *http.Client
	URL     func(query string, from, size int) string
	Parse   func(body []byte) (domain.PaginaVitrine, error)
	PageSize int
	// Headers optional request headers (e.g. Fort search token).
	Headers func(ctx context.Context) (http.Header, error)
}

func (c Client) Buscar(ctx context.Context, query string) (domain.ResultadoVitrine, error) {
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	size := c.PageSize
	if size <= 0 {
		size = 12
	}
	var all []domain.CartaoVitrine
	from := 0
	for {
		pagina, err := c.fetchPage(ctx, httpClient, query, from, size)
		if err != nil {
			if len(all) > 0 {
				return domain.ResultadoVitrine{Cartoes: all, Esgotada: false}, nil
			}
			return domain.ResultadoVitrine{}, err
		}
		all = append(all, pagina.Cartoes...)
		if !pagina.HasNext || len(pagina.Cartoes) == 0 {
			return domain.ResultadoVitrine{Cartoes: all, Esgotada: true}, nil
		}
		if pagina.NextFrom > from {
			from = pagina.NextFrom
		} else {
			from += size
		}
	}
}

func (c Client) fetchPage(ctx context.Context, httpClient *http.Client, query string, from, size int) (domain.PaginaVitrine, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL(query, from, size), nil)
	if err != nil {
		return domain.PaginaVitrine{}, err
	}
	req.Header.Set("User-Agent", "ofertas-scraper-v2/1.0")
	if c.Headers != nil {
		h, err := c.Headers(ctx)
		if err != nil {
			return domain.PaginaVitrine{}, err
		}
		for k, vs := range h {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return domain.PaginaVitrine{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.PaginaVitrine{}, err
	}
	if resp.StatusCode >= 400 {
		return domain.PaginaVitrine{}, fmt.Errorf("vitrine HTTP %d", resp.StatusCode)
	}
	return c.Parse(body)
}
