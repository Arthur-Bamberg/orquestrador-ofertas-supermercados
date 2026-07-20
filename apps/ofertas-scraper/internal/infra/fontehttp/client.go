package fontehttp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

// Matches absolute or root-relative URLs ending in .pdf (case-insensitive).
// Body is pre-normalized so JSON escapes (\/) become /.
var pdfURLRe = regexp.MustCompile(`(?i)(?:https?://[^\s"'<>]+|/[^\s"'<>]+)\.pdf`)

// Bare filename.pdf inside JSON (e.g. Via "arquivo" field).
var pdfFilenameRe = regexp.MustCompile(`(?i)([A-Za-z0-9][A-Za-z0-9._-]*%?)*\.pdf`)

const browserUA = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

// Client discovers PDF links from Fonte page bodies or JSON APIs (ADR 0020).
type Client struct {
	HTTP *http.Client
}

func New(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{HTTP: httpClient}
}

func (c *Client) DiscoverPDFs(ctx context.Context, fonte domain.Fonte) ([]domain.PDFDescoberto, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fonte.URL, nil)
	if err != nil {
		return nil, err
	}
	setBrowserHeaders(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("fonte http %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	base, err := url.Parse(fonte.URL)
	if err != nil {
		return nil, err
	}
	ct := res.Header.Get("Content-Type")
	return extractPDFLinks(string(body), base, ct), nil
}

func (c *Client) DownloadPDF(ctx context.Context, pdfURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pdfURL, nil)
	if err != nil {
		return nil, err
	}
	setBrowserHeaders(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("download pdf http %d", res.StatusCode)
	}
	return io.ReadAll(res.Body)
}

func setBrowserHeaders(req *http.Request) {
	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,application/json,*/*;q=0.8")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
}

func extractPDFLinks(body string, base *url.URL, contentType string) []domain.PDFDescoberto {
	normalized := strings.ReplaceAll(body, `\/`, `/`)
	seen := map[string]struct{}{}
	var out []domain.PDFDescoberto

	add := func(raw string) {
		raw = strings.TrimRight(raw, `.,);]}`)
		u, err := resolvePDFRef(base, raw)
		if err != nil {
			return
		}
		if !strings.HasSuffix(strings.ToLower(u.Path), ".pdf") {
			return
		}
		abs := u.String()
		if _, ok := seen[abs]; ok {
			return
		}
		seen[abs] = struct{}{}
		out = append(out, domain.PDFDescoberto{
			Filename: path.Base(u.Path),
			URL:      abs,
		})
	}

	for _, match := range pdfURLRe.FindAllString(normalized, -1) {
		add(match)
	}

	// JSON APIs (e.g. Via /api/ofertas) often return bare "arquivo":"....pdf"
	if looksLikeJSON(contentType, normalized) {
		for _, name := range collectJSONPDFFilenames(normalized) {
			add(name)
		}
	}
	return out
}

func looksLikeJSON(contentType, body string) bool {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "json") {
		return true
	}
	trim := strings.TrimSpace(body)
	return strings.HasPrefix(trim, "[") || strings.HasPrefix(trim, "{")
}

func collectJSONPDFFilenames(body string) []string {
	var v any
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		// fallback: regex bare filenames in text
		return pdfFilenameRe.FindAllString(body, -1)
	}
	var out []string
	walkJSON(v, &out)
	return out
}

func walkJSON(v any, out *[]string) {
	switch t := v.(type) {
	case map[string]any:
		for _, child := range t {
			walkJSON(child, out)
		}
	case []any:
		for _, child := range t {
			walkJSON(child, out)
		}
	case string:
		s := strings.TrimSpace(t)
		if strings.HasSuffix(strings.ToLower(s), ".pdf") {
			*out = append(*out, s)
		}
	}
}

// resolvePDFRef turns absolute/relative URLs or bare filenames into absolute URLs.
// Bare filenames from /.../api/... endpoints resolve to /.../encarte/{file} (Via pattern).
func resolvePDFRef(base *url.URL, ref string) (*url.URL, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("empty")
	}
	if strings.Contains(ref, "://") || strings.HasPrefix(ref, "/") {
		return base.Parse(ref)
	}
	// bare filename
	if i := strings.Index(base.Path, "/api/"); i >= 0 {
		encarte := &url.URL{
			Scheme: base.Scheme,
			Host:   base.Host,
			Path:   path.Join(base.Path[:i], "encarte", path.Base(ref)),
		}
		return encarte, nil
	}
	return base.Parse(ref)
}
