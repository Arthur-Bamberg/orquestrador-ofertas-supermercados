package extrator

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"

	_ "embed"
)

//go:embed cursor_bridge.py
var cursorBridgePy []byte

const (
	defaultCursorModel  = "composer-2.5"
	defaultCursorPython = "python3"
)

// CursorConfig wires the Cursor Agent SDK Extrator adapter (ADR 0034).
type CursorConfig struct {
	APIKey             string
	Model              string
	PythonPath         string
	PromptPath         string
	ExtracaoSchemaPath string
	OfertaSchemaPath   string
	Logger             *log.Logger
}

// Cursor implements domain.Extrator via the Cursor local Agent SDK (Python bridge).
type Cursor struct {
	apiKey     string
	model      string
	python     string
	prompt     string
	schemaJSON string
	bridgePath string
	log        *log.Logger
}

type cursorBridgeRequest struct {
	APIKey      string               `json:"apiKey"`
	Model       string               `json:"model"`
	ModelParams []cursorModelParam   `json:"modelParams"`
	Prompt      string               `json:"prompt"`
	SchemaJSON  string               `json:"schemaJSON"`
	Pages       []cursorBridgePage   `json:"pages"`
}

type cursorModelParam struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type cursorBridgePage struct {
	Page        int    `json:"page"`
	JPEGBase64  string `json:"jpegBase64"`
}

type cursorBridgeResponse struct {
	OK      bool `json:"ok"`
	Ofertas []domain.CandidatoOferta `json:"ofertas"`
	Uso     *struct {
		Model        string `json:"model"`
		PromptTokens int64  `json:"promptTokens"`
		CacheTokens  int64  `json:"cacheTokens"`
		OutputTokens int64  `json:"outputTokens"`
		Paginas      []struct {
			Page         int   `json:"page"`
			PromptTokens int64 `json:"promptTokens"`
			CacheTokens  int64 `json:"cacheTokens"`
			OutputTokens int64 `json:"outputTokens"`
		} `json:"paginas"`
	} `json:"uso"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewCursor loads prompt/schema and materializes the embedded Python bridge.
func NewCursor(cfg CursorConfig) (*Cursor, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("CURSOR_API_KEY required")
	}
	model := cfg.Model
	if model == "" {
		model = defaultCursorModel
	}
	python := cfg.PythonPath
	if python == "" {
		python = envOrDefault("CURSOR_PYTHON", defaultCursorPython)
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

	_, schemaJSON, err := loadGeminiResponseSchema(extracaoPath, ofertaPath)
	if err != nil {
		return nil, err
	}

	bridgePath, err := materializeCursorBridge()
	if err != nil {
		return nil, err
	}

	return &Cursor{
		apiKey:     cfg.APIKey,
		model:      model,
		python:     python,
		prompt:     prompt,
		schemaJSON: schemaJSON,
		bridgePath: bridgePath,
		log:        logger,
	}, nil
}

func envOrDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func materializeCursorBridge() (string, error) {
	dir := filepath.Join(os.TempDir(), "ofertas-scraper-cursor-bridge")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("cursor bridge dir: %w", err)
	}
	path := filepath.Join(dir, "cursor_bridge.py")
	if err := os.WriteFile(path, cursorBridgePy, 0o644); err != nil {
		return "", fmt.Errorf("write cursor bridge: %w", err)
	}
	return path, nil
}

func (c *Cursor) Extract(ctx context.Context, images []domain.PageImage) ([]domain.CandidatoOferta, []byte, *domain.UsoExtrator, error) {
	if len(images) == 0 {
		return nil, nil, nil, fmt.Errorf("cursor extrator: no images")
	}

	// One Agent SDK turn per page (ADR 0031), one durable agent per Extract call.
	var all []domain.CandidatoOferta
	uso := &domain.UsoExtrator{Provider: domain.ExtratorProviderCursor, Model: c.model}
	for _, img := range images {
		if len(img.JPEG) == 0 {
			continue
		}
		cands, pageUso, err := c.extractPages(ctx, []domain.PageImage{img})
		if err != nil {
			return nil, nil, nil, err
		}
		if pageUso != nil {
			uso.PromptTokens += pageUso.PromptTokens
			uso.CacheTokens += pageUso.CacheTokens
			uso.OutputTokens += pageUso.OutputTokens
			uso.Paginas = append(uso.Paginas, pageUso.Paginas...)
		}
		page := img.Page
		if page <= 0 {
			page = 1
		}
		c.log.Printf("cursor extrator page=%d ofertas=%d promptTokens=%d cacheTokens=%d outputTokens=%d",
			page, len(cands),
			pageUsoToken(firstPageUso(pageUso), func(u domain.UsoExtratorPagina) int64 { return u.PromptTokens }),
			pageUsoToken(firstPageUso(pageUso), func(u domain.UsoExtratorPagina) int64 { return u.CacheTokens }),
			pageUsoToken(firstPageUso(pageUso), func(u domain.UsoExtratorPagina) int64 { return u.OutputTokens }),
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
		return nil, nil, nil, fmt.Errorf("cursor extrator: marshal merged raw: %w", err)
	}
	c.log.Printf("cursor extrator total ofertas=%d promptTokens=%d cacheTokens=%d outputTokens=%d",
		len(all), uso.PromptTokens, uso.CacheTokens, uso.OutputTokens)
	return all, raw, uso, nil
}

func firstPageUso(u *domain.UsoExtrator) *domain.UsoExtratorPagina {
	if u == nil || len(u.Paginas) == 0 {
		return nil
	}
	p := u.Paginas[0]
	return &p
}

func (c *Cursor) extractPages(ctx context.Context, images []domain.PageImage) ([]domain.CandidatoOferta, *domain.UsoExtrator, error) {
	pages := make([]cursorBridgePage, 0, len(images))
	for _, img := range images {
		page := img.Page
		if page <= 0 {
			page = 1
		}
		pages = append(pages, cursorBridgePage{
			Page:       page,
			JPEGBase64: base64.StdEncoding.EncodeToString(img.JPEG),
		})
	}

	req := cursorBridgeRequest{
		APIKey: c.apiKey,
		Model:  c.model,
		ModelParams: []cursorModelParam{
			{ID: "fast", Value: "true"},
		},
		Prompt:     c.prompt,
		SchemaJSON: c.schemaJSON,
		Pages:      pages,
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("cursor extrator: marshal request: %w", err)
	}

	cmd := exec.CommandContext(ctx, c.python, c.bridgePath)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Ensure user-local cursor-sdk scripts are visible when PATH is minimal.
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	if home, err := os.UserHomeDir(); err == nil {
		localBin := filepath.Join(home, ".local", "bin")
		cmd.Env = append(cmd.Env, "PATH="+localBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	}

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, nil, fmt.Errorf("%w: %v", domain.ErrExtratorIndisponivel, ctx.Err())
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, nil, mapCursorBridgeError("bridge process failed: "+msg, "")
	}

	var resp cursorBridgeResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, nil, fmt.Errorf("cursor extrator: decode bridge response: %w (stderr=%s)", err, strings.TrimSpace(stderr.String()))
	}
	if !resp.OK {
		code, msg := "", "unknown bridge error"
		if resp.Error != nil {
			code = resp.Error.Code
			if resp.Error.Message != "" {
				msg = resp.Error.Message
			}
		}
		return nil, nil, mapCursorBridgeError(msg, code)
	}
	cands := resp.Ofertas
	if cands == nil {
		cands = []domain.CandidatoOferta{}
	}
	var uso *domain.UsoExtrator
	if resp.Uso != nil {
		uso = &domain.UsoExtrator{
			Model:        resp.Uso.Model,
			PromptTokens: resp.Uso.PromptTokens,
			CacheTokens:  resp.Uso.CacheTokens,
			OutputTokens: resp.Uso.OutputTokens,
		}
		if uso.Model == "" {
			uso.Model = c.model
		}
		for _, p := range resp.Uso.Paginas {
			uso.Paginas = append(uso.Paginas, domain.UsoExtratorPagina{
				Page:         p.Page,
				PromptTokens: p.PromptTokens,
				CacheTokens:  p.CacheTokens,
				OutputTokens: p.OutputTokens,
			})
		}
	}
	return cands, uso, nil
}

func mapCursorBridgeError(msg, code string) error {
	code = strings.ToLower(strings.TrimSpace(code))
	lower := strings.ToLower(msg)
	cota := code == "rate_limit" ||
		strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "429") ||
		strings.Contains(lower, "resource_exhausted")
	if cota {
		return fmt.Errorf("%w: %s", domain.ErrExtratorCota, msg)
	}
	indisponivel := code == "unavailable" ||
		strings.Contains(lower, "unavailable") ||
		strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "deadline")
	if indisponivel {
		return fmt.Errorf("%w: %s", domain.ErrExtratorIndisponivel, msg)
	}
	return fmt.Errorf("cursor extrator: %s", msg)
}
