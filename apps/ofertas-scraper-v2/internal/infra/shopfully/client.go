package shopfully

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
)

const (
	defaultSite        = "https://www.shopfully.com.br"
	defaultPublication = "https://api-viewer-zmags.shopfully.cloud"
	browserUA          = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
)

// Config points at Shopfully site and publication API (overridable in tests).
type Config struct {
	SiteBase        string
	PublicationBase string
}

// Client fetches listing, viewer, publication pages and JPEG bytes.
type Client struct {
	HTTP            *http.Client
	SiteBase        string
	PublicationBase string
}

func New(httpClient *http.Client, cfg Config) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	site := strings.TrimRight(cfg.SiteBase, "/")
	if site == "" {
		site = defaultSite
	}
	pub := strings.TrimRight(cfg.PublicationBase, "/")
	if pub == "" {
		pub = defaultPublication
	}
	return &Client{HTTP: httpClient, SiteBase: site, PublicationBase: pub}
}

func (c *Client) Listagem(ctx context.Context, path string) ([]domain.Encarte, error) {
	body, _, err := c.get(ctx, c.SiteBase+path)
	if err != nil {
		return nil, err
	}
	return ParseListagem(string(body))
}

func (c *Client) Paginas(ctx context.Context, encarte domain.Encarte) ([]domain.Pagina, error) {
	viewerURL, err := url.JoinPath(c.SiteBase, encarte.ViewerPath)
	if err != nil {
		return nil, err
	}
	if q := strings.Index(encarte.ViewerPath, "?"); q >= 0 {
		base, _ := url.JoinPath(c.SiteBase, encarte.ViewerPath[:q])
		viewerURL = base + encarte.ViewerPath[q:]
	}
	htmlBody, _, err := c.get(ctx, viewerURL)
	if err != nil {
		return nil, fmt.Errorf("viewer %s: %w", encarte.ID, err)
	}
	pubID, err := ParsePublicationID(string(htmlBody))
	if err != nil {
		return nil, fmt.Errorf("viewer %s: %w", encarte.ID, err)
	}
	pubRaw, _, err := c.get(ctx, c.PublicationBase+"/publication/"+pubID)
	if err != nil {
		return nil, fmt.Errorf("publication %s: %w", pubID, err)
	}
	firstBundle, err := PublicationBundlePath(pubRaw)
	if err != nil {
		return nil, err
	}
	bundleRaw, _, err := c.get(ctx, firstBundle)
	if err != nil {
		return nil, fmt.Errorf("bundle %s: %w", firstBundle, err)
	}
	bundles, err := BundlePaths(bundleRaw)
	if err != nil {
		return nil, err
	}
	byNumero := map[int]domain.Pagina{}
	for _, bundleURL := range bundles {
		raw := bundleRaw
		if bundleURL != firstBundle {
			raw, _, err = c.get(ctx, bundleURL)
			if err != nil {
				return nil, fmt.Errorf("bundle %s: %w", bundleURL, err)
			}
		}
		pages, err := BestPageImages(raw)
		if err != nil {
			return nil, err
		}
		for _, p := range pages {
			byNumero[p.Numero] = p
		}
	}
	out := make([]domain.Pagina, 0, len(byNumero))
	for _, p := range byNumero {
		out = append(out, p)
	}
	return sortPaginas(out), nil
}

func (c *Client) Download(ctx context.Context, rawURL string) ([]byte, error) {
	body, ct, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(strings.ToLower(ct), "image/") {
		return nil, fmt.Errorf("nao e imagem: %s (%s)", rawURL, ct)
	}
	return body, nil
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Accept", "*/*")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, "", fmt.Errorf("http %d %s", res.StatusCode, rawURL)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}
	return body, res.Header.Get("Content-Type"), nil
}
