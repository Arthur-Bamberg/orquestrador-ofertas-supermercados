package vitrinehttp

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/domain"
)

type Client struct {
	HTTP  *http.Client
	URL   func(query string) string
	Parse func(body []byte) ([]domain.CartaoVitrine, error)
}

func (c Client) Buscar(ctx context.Context, query string) ([]domain.CartaoVitrine, error) {
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL(query), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ofertas-web-scraper/1.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("vitrine HTTP %d", resp.StatusCode)
	}
	return c.Parse(body)
}
