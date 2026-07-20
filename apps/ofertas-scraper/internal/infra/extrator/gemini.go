package extrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"

	"google.golang.org/genai"
)

const (
	defaultGeminiModel = "gemini-3-flash-preview"
	defaultCacheTTL    = time.Hour
)

// GeminiConfig wires the Gemini Extrator adapter (ADR 0023).
type GeminiConfig struct {
	APIKey            string
	Model             string
	PromptPath        string
	ExtracaoSchemaPath string
	OfertaSchemaPath  string
	CacheTTL          time.Duration
	Logger            *log.Logger
}

// Gemini implements domain.Extrator via generateContent + optional context cache.
type Gemini struct {
	client     *genai.Client
	model      string
	prompt     string
	schema     any
	schemaJSON string
	cacheTTL   time.Duration
	log        *log.Logger

	mu        sync.Mutex
	cacheName string
}

// NewGemini loads prompt/schema, creates the API client, and best-effort creates a
// context cache for the stable system prompt (+ schema text). Documento images are
// never cached.
func NewGemini(ctx context.Context, cfg GeminiConfig) (*Gemini, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY required")
	}
	model := cfg.Model
	if model == "" {
		model = defaultGeminiModel
	}
	promptPath := cfg.PromptPath
	if promptPath == "" {
		promptPath = "./prompts/extrator.txt"
	}
	extracaoPath := cfg.ExtracaoSchemaPath
	if extracaoPath == "" {
		extracaoPath = "./schemas/extracao.json"
	}
	ofertaPath := cfg.OfertaSchemaPath
	if ofertaPath == "" {
		ofertaPath = "./schemas/oferta.json"
	}
	ttl := cfg.CacheTTL
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}

	promptBytes, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, fmt.Errorf("read extrator prompt: %w", err)
	}
	prompt := strings.TrimSpace(string(promptBytes))
	if prompt == "" {
		return nil, fmt.Errorf("extrator prompt is empty")
	}

	schema, schemaJSON, err := loadGeminiResponseSchema(extracaoPath, ofertaPath)
	if err != nil {
		return nil, err
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini client: %w", err)
	}

	g := &Gemini{
		client:     client,
		model:      model,
		prompt:     prompt,
		schema:     schema,
		schemaJSON: schemaJSON,
		cacheTTL:   ttl,
		log:        logger,
	}
	if err := g.ensureCache(ctx); err != nil {
		logger.Printf("gemini context cache unavailable; using per-request system instruction: %v", err)
	}
	return g, nil
}

func (g *Gemini) Extract(ctx context.Context, images []domain.PageImage) ([]domain.CandidatoOferta, []byte, *domain.UsoExtrator, error) {
	if len(images) == 0 {
		return nil, nil, nil, fmt.Errorf("gemini extrator: no images")
	}
	// One API call per page (ADR 0031): dense encartes lose Ofertas when many pages
	// share a single vision pass.
	var all []domain.CandidatoOferta
	uso := &domain.UsoExtrator{Model: g.model}
	for _, img := range images {
		if len(img.JPEG) == 0 {
			continue
		}
		cands, _, pageUso, err := g.extract(ctx, img, true)
		if err != nil {
			return nil, nil, nil, err
		}
		if pageUso != nil {
			uso.PromptTokens += pageUso.PromptTokens
			uso.CacheTokens += pageUso.CacheTokens
			uso.OutputTokens += pageUso.OutputTokens
			uso.Paginas = append(uso.Paginas, *pageUso)
		}
		g.log.Printf("gemini extrator page=%d ofertas=%d promptTokens=%d cacheTokens=%d outputTokens=%d",
			img.Page, len(cands),
			pageUsoToken(pageUso, func(u domain.UsoExtratorPagina) int64 { return u.PromptTokens }),
			pageUsoToken(pageUso, func(u domain.UsoExtratorPagina) int64 { return u.CacheTokens }),
			pageUsoToken(pageUso, func(u domain.UsoExtratorPagina) int64 { return u.OutputTokens }),
		)
		all = append(all, cands...)
	}
	if all == nil {
		all = []domain.CandidatoOferta{}
	}
	raw, err := json.Marshal(struct {
		Ofertas []domain.CandidatoOferta `json:"ofertas"`
	}{Ofertas: all})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gemini extrator: marshal merged raw: %w", err)
	}
	g.log.Printf("gemini extrator total ofertas=%d promptTokens=%d cacheTokens=%d outputTokens=%d",
		len(all), uso.PromptTokens, uso.CacheTokens, uso.OutputTokens)
	return all, raw, uso, nil
}

func pageUsoToken(u *domain.UsoExtratorPagina, f func(domain.UsoExtratorPagina) int64) int64 {
	if u == nil {
		return 0
	}
	return f(*u)
}

func (g *Gemini) extract(ctx context.Context, img domain.PageImage, retryCacheMiss bool) ([]domain.CandidatoOferta, []byte, *domain.UsoExtratorPagina, error) {
	if len(img.JPEG) == 0 {
		return nil, nil, nil, fmt.Errorf("gemini extrator: empty page image")
	}

	pageLabel := img.Page
	if pageLabel <= 0 {
		pageLabel = 1
	}
	parts := []*genai.Part{
		{Text: fmt.Sprintf(
			"Extraia TODAS as ofertas com preço legível desta única página (página %d) do encarte. Não omita itens da grade; não amostrar.",
			pageLabel,
		)},
		genai.NewPartFromBytes(img.JPEG, "image/jpeg"),
	}

	g.mu.Lock()
	cacheName := g.cacheName
	g.mu.Unlock()

	cfg := &genai.GenerateContentConfig{
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: g.schema,
		Temperature:        genai.Ptr(float32(0.2)),
	}
	if cacheName != "" {
		cfg.CachedContent = cacheName
	} else {
		cfg.SystemInstruction = genai.NewContentFromText(g.prompt, genai.RoleUser)
	}

	contents := []*genai.Content{
		{Role: genai.RoleUser, Parts: parts},
	}

	resp, err := g.client.Models.GenerateContent(ctx, g.model, contents, cfg)
	if err != nil {
		if retryCacheMiss && cacheName != "" && isCacheMiss(err) {
			g.mu.Lock()
			g.cacheName = ""
			g.mu.Unlock()
			_ = g.ensureCache(ctx)
			return g.extract(ctx, img, false)
		}
		return nil, nil, nil, mapGeminiError(err)
	}

	rawText := strings.TrimSpace(resp.Text())
	if rawText == "" {
		return nil, nil, nil, fmt.Errorf("gemini extrator: empty response")
	}
	raw := []byte(rawText)

	var pageUso *domain.UsoExtratorPagina
	if resp.UsageMetadata != nil {
		page := img.Page
		if page <= 0 {
			page = 1
		}
		pageUso = &domain.UsoExtratorPagina{
			Page:         page,
			PromptTokens: int64(resp.UsageMetadata.PromptTokenCount),
			CacheTokens:  int64(resp.UsageMetadata.CachedContentTokenCount),
			OutputTokens: int64(resp.UsageMetadata.CandidatesTokenCount),
		}
	}

	var parsed struct {
		Ofertas []domain.CandidatoOferta `json:"ofertas"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, raw, pageUso, fmt.Errorf("gemini extrator: decode structured output: %w", err)
	}
	if parsed.Ofertas == nil {
		parsed.Ofertas = []domain.CandidatoOferta{}
	}
	return parsed.Ofertas, raw, pageUso, nil
}

func (g *Gemini) ensureCache(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.cacheName != "" {
		return nil
	}

	cache, err := g.client.Caches.Create(ctx, g.model, &genai.CreateCachedContentConfig{
		DisplayName: "ofertas-scraper-extrator",
		TTL:         g.cacheTTL,
		SystemInstruction: genai.NewContentFromText(g.prompt, genai.RoleUser),
		// Schema text bulks the cache toward model minimums; images stay per-request.
		Contents: []*genai.Content{
			genai.NewContentFromText(
				"Contrato de saída (JSON Schema). Responda sempre neste formato:\n"+g.schemaJSON,
				genai.RoleUser,
			),
		},
	})
	if err != nil {
		return err
	}
	if cache == nil || cache.Name == "" {
		return fmt.Errorf("cache created without name")
	}
	g.cacheName = cache.Name
	g.log.Printf("gemini context cache ready: %s (ttl=%s)", g.cacheName, g.cacheTTL)
	return nil
}

func mapGeminiError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %v", domain.ErrExtratorIndisponivel, err)
	}
	var apiErr genai.APIError
	if errors.As(err, &apiErr) {
		if apiErr.Code == 429 || apiErr.Code >= 500 {
			return fmt.Errorf("%w: %v", domain.ErrExtratorIndisponivel, err)
		}
		status := strings.ToUpper(apiErr.Status)
		if status == "UNAVAILABLE" || status == "RESOURCE_EXHAUSTED" || status == "DEADLINE_EXCEEDED" {
			return fmt.Errorf("%w: %v", domain.ErrExtratorIndisponivel, err)
		}
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unavailable") ||
		strings.Contains(msg, "resource_exhausted") ||
		strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "429") ||
		strings.Contains(msg, "500") ||
		strings.Contains(msg, "502") ||
		strings.Contains(msg, "503") {
		return fmt.Errorf("%w: %v", domain.ErrExtratorIndisponivel, err)
	}
	return err
}

func isCacheMiss(err error) bool {
	if err == nil {
		return false
	}
	var apiErr genai.APIError
	if errors.As(err, &apiErr) && apiErr.Code == 404 {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "cachedcontent") && strings.Contains(msg, "not found")
}
