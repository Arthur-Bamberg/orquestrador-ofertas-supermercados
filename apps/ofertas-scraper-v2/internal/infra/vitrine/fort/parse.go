package fort

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrine/osuper"
)

const (
	BaseURL       = "https://www.fortatacadista.com.br"
	AccountID     = "334"
	LojaPadrao    = "115"  // display code on the site (Balneário Camboriú)
	LojaSenseID   = "1585" // Sense/Osuper store id for that default loja
	SearchTokenHeader = "X-Osuper-Search-Token"
)

var tokenRE = regexp.MustCompile(`searchEngineToken":"([^"]+)"`)

func SearchURL(query string, from, size int) string {
	return osuper.SearchURL(AccountID, LojaSenseID, query, from, size)
}

func ParseSearch(body []byte) (domain.PaginaVitrine, error) {
	return osuper.ParseSearch(body)
}

// TokenProvider fetches and caches the Sense search token from the Fort home page.
type TokenProvider struct {
	HTTP    *http.Client
	HomeURL string

	mu    sync.Mutex
	token string
	until time.Time
}

func (p *TokenProvider) Header(ctx context.Context) (http.Header, error) {
	tok, err := p.Token(ctx)
	if err != nil {
		return nil, err
	}
	h := make(http.Header)
	h.Set(SearchTokenHeader, tok)
	return h, nil
}

func (p *TokenProvider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.token != "" && time.Now().Before(p.until) {
		return p.token, nil
	}
	httpClient := p.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	home := p.HomeURL
	if home == "" {
		home = BaseURL + "/"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, home, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ofertas-scraper-v2/1.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("fort home HTTP %d", resp.StatusCode)
	}
	m := tokenRE.FindSubmatch(body)
	if m == nil {
		return "", fmt.Errorf("fort searchEngineToken not found")
	}
	p.token = string(m[1])
	p.until = time.Now().Add(30 * time.Minute)
	return p.token, nil
}
