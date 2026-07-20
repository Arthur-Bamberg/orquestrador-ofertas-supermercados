package presentation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/artefato"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/extrator"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/filenamedate"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/fontehttp"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/raster"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/upstash"
)

// Env holds process configuration.
type Env struct {
	UpstashURL         string
	UpstashToken       string
	ArtefatoRoot       string
	RasterMaxPx        int
	RasterJPEGQ        int
	SeedPath           string
	UseStubExtrator    bool
	GeminiAPIKey       string
	GeminiModel        string
	ExtratorPromptPath string
	ExtracaoSchemaPath string
	OfertaSchemaPath   string
	RunFonteID         string
	RunMaxDocumentos   int
}

func LoadEnv() Env {
	maxPx, _ := strconv.Atoi(os.Getenv("RASTER_MAX_EDGE_PX"))
	jpegQ, _ := strconv.Atoi(os.Getenv("RASTER_JPEG_QUALITY"))
	maxDocs, _ := strconv.Atoi(os.Getenv("RUN_MAX_DOCUMENTOS"))
	apiKey := os.Getenv("GEMINI_API_KEY")
	stub := os.Getenv("EXTRATOR_STUB") == "1" || apiKey == ""
	return Env{
		UpstashURL:         os.Getenv("UPSTASH_REDIS_REST_URL"),
		UpstashToken:       os.Getenv("UPSTASH_REDIS_REST_TOKEN"),
		ArtefatoRoot:       envOr("ARTEFATO_ROOT", "./.data/artefatos"),
		RasterMaxPx:        maxPx,
		RasterJPEGQ:        jpegQ,
		SeedPath:           envOr("SEED_PATH", "./seed/fontes.json"),
		UseStubExtrator:    stub,
		GeminiAPIKey:       apiKey,
		GeminiModel:        os.Getenv("GEMINI_MODEL"),
		ExtratorPromptPath: envOr("EXTRATOR_PROMPT_PATH", "./prompts/extrator.txt"),
		ExtracaoSchemaPath: envOr("EXTRACAO_SCHEMA_PATH", "./schemas/extracao.json"),
		OfertaSchemaPath:   envOr("OFERTA_SCHEMA_PATH", "./schemas/oferta.json"),
		RunFonteID:         os.Getenv("RUN_FONTE_ID"),
		RunMaxDocumentos:   maxDocs,
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type seedFile struct {
	Mercados []domain.Mercado `json:"mercados"`
	Fontes   []domain.Fonte   `json:"fontes"`
}

// RunSeed upserts Mercados/Fontes from seed JSON into Redis.
func RunSeed(ctx context.Context, env Env) error {
	if env.UpstashURL == "" || env.UpstashToken == "" {
		return fmt.Errorf("UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN required")
	}
	raw, err := os.ReadFile(env.SeedPath)
	if err != nil {
		return err
	}
	var seed seedFile
	if err := json.Unmarshal(raw, &seed); err != nil {
		return err
	}
	client := upstash.NewClient(env.UpstashURL, env.UpstashToken, nil)
	mercados := upstash.NewMercadoRepo(client)
	fontes := upstash.NewFonteRepo(client)
	for _, m := range seed.Mercados {
		if m.ID == "" {
			m.ID = domain.MercadoID(application.NewID())
		}
		if err := mercados.Save(ctx, m); err != nil {
			return err
		}
		log.Printf("seed mercado %s (%s)", m.ID, m.Nome)
	}
	for _, f := range seed.Fontes {
		if f.ID == "" {
			f.ID = domain.FonteID(application.NewID())
		}
		if err := fontes.Save(ctx, f); err != nil {
			return err
		}
		log.Printf("seed fonte %s → %s", f.ID, f.URL)
	}
	return nil
}

// RunDaily wires adapters and runs the daily job.
func RunDaily(ctx context.Context, env Env) error {
	if env.UpstashURL == "" || env.UpstashToken == "" {
		return fmt.Errorf("UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN required")
	}
	client := upstash.NewClient(env.UpstashURL, env.UpstashToken, nil)

	var ext domain.Extrator
	if env.UseStubExtrator {
		log.Printf("using Extrator stub (EXTRATOR_STUB=1 or GEMINI_API_KEY empty)")
		ext = extrator.Stub{Candidatos: nil}
	} else {
		g, err := extrator.NewGemini(ctx, extrator.GeminiConfig{
			APIKey:             env.GeminiAPIKey,
			Model:              env.GeminiModel,
			PromptPath:         env.ExtratorPromptPath,
			ExtracaoSchemaPath: env.ExtracaoSchemaPath,
			OfertaSchemaPath:   env.OfertaSchemaPath,
			Logger:             log.Default(),
		})
		if err != nil {
			return fmt.Errorf("gemini extrator: %w", err)
		}
		model := env.GeminiModel
		if model == "" {
			model = "gemini-3-flash-preview"
		}
		log.Printf("using Gemini Extrator model=%s", model)
		ext = g
	}

	if env.RunFonteID != "" {
		log.Printf("smoke: RUN_FONTE_ID=%s", env.RunFonteID)
	}
	if env.RunMaxDocumentos > 0 {
		log.Printf("smoke: RUN_MAX_DOCUMENTOS=%d", env.RunMaxDocumentos)
	}

	deps := application.RunDailyJobDeps{
		Fontes:     upstash.NewFonteRepo(client),
		Documentos: upstash.NewDocumentoRepo(client),
		Produtos:   upstash.NewProdutoRepo(client),
		Marcas:     upstash.NewMarcaRepo(client),
		Ofertas:    upstash.NewOfertaRepo(client),
		Falhas:     upstash.NewFalhaRepo(client),
		FonteHTTP:  fontehttp.New(http.DefaultClient),
		Raster:     raster.NewPdftoppm(env.RasterMaxPx, env.RasterJPEGQ),
		Extrator:   ext,
		Artefatos:  artefato.NewLocalStore(env.ArtefatoRoot),
		Dates:      filenamedate.Parser{},
		Log:        log.Default(),
		ExtratorRetryBudget: time.Hour,
		OnlyFonteID:         domain.FonteID(env.RunFonteID),
		MaxDocumentos:       env.RunMaxDocumentos,
	}
	return application.RunDailyJob(ctx, deps)
}
