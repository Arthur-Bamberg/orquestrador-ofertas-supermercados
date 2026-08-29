package presentation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/artefato"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/extrator"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/filenamedate"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/fontehttp"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/raster"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store/redismigrate"
)

// Env holds process configuration.
type Env struct {
	DatabaseURL        string
	ArtefatoRoot       string
	RasterMaxPx        int
	RasterJPEGQ        int
	SeedPath           string
	UseStubExtrator    bool
	ExtratorProvider   string // auto|gemini|cursor (ADR 0034)
	GeminiAPIKey       string
	GeminiModel        string
	CursorAPIKey       string
	CursorModel        string
	CursorPython       string
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
	geminiKey := os.Getenv("GEMINI_API_KEY")
	cursorKey := os.Getenv("CURSOR_API_KEY")
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("EXTRATOR_PROVIDER")))
	stub := os.Getenv("EXTRATOR_STUB") == "1" || (geminiKey == "" && cursorKey == "")
	return Env{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		ArtefatoRoot:       envOr("ARTEFATO_ROOT", "./.data/artefatos"),
		RasterMaxPx:        maxPx,
		RasterJPEGQ:        jpegQ,
		SeedPath:           envOr("SEED_PATH", "./seed/fontes.json"),
		UseStubExtrator:    stub,
		ExtratorProvider:   provider,
		GeminiAPIKey:       geminiKey,
		GeminiModel:        os.Getenv("GEMINI_MODEL"),
		CursorAPIKey:       cursorKey,
		CursorModel:        os.Getenv("CURSOR_MODEL"),
		CursorPython:       os.Getenv("CURSOR_PYTHON"),
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

func openCatalog(ctx context.Context, env Env) (*store.Catalog, error) {
	if env.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL required")
	}
	return store.Open(ctx, env.DatabaseURL)
}

// RunSeed upserts Mercados/Fontes from seed JSON into the catalog.
func RunSeed(ctx context.Context, env Env) error {
	catalog, err := openCatalog(ctx, env)
	if err != nil {
		return err
	}
	defer catalog.Close()
	raw, err := os.ReadFile(env.SeedPath)
	if err != nil {
		return err
	}
	var seed seedFile
	if err := json.Unmarshal(raw, &seed); err != nil {
		return err
	}
	mercados := store.NewMercadoRepo(catalog)
	fontes := store.NewFonteRepo(catalog)
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

func RunDiscover(ctx context.Context, env Env, fonteID string) error {
	catalog, err := openCatalog(ctx, env)
	if err != nil {
		return err
	}
	defer catalog.Close()
	deps := application.RunDailyJobDeps{
		Fontes:     store.NewFonteRepo(catalog),
		Documentos: store.NewDocumentoRepo(catalog),
		FonteHTTP:  fontehttp.New(http.DefaultClient),
		Log:        log.Default(),
	}
	return application.DiscoverDocumentos(ctx, deps, domain.FonteID(fonteID))
}

func RunMigrateFromRedis(ctx context.Context, env Env, redisURL, redisToken string) error {
	catalog, err := openCatalog(ctx, env)
	if err != nil {
		return err
	}
	defer catalog.Close()
	if redisURL == "" || redisToken == "" {
		return fmt.Errorf("UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN required for migrate-from-redis")
	}
	src := redismigrate.NewClient(redisURL, redisToken, nil)
	return redismigrate.CopyFromRedis(ctx, src, catalog)
}

// RunDaily wires adapters and runs the daily job.
func RunDaily(ctx context.Context, env Env) error {
	catalog, err := openCatalog(ctx, env)
	if err != nil {
		return err
	}
	defer catalog.Close()
	deps, err := buildRunDeps(ctx, env, catalog)
	if err != nil {
		return err
	}
	if env.RunFonteID != "" {
		log.Printf("smoke: RUN_FONTE_ID=%s", env.RunFonteID)
	}
	if env.RunMaxDocumentos > 0 {
		log.Printf("smoke: RUN_MAX_DOCUMENTOS=%d", env.RunMaxDocumentos)
	}
	return application.RunDailyJob(ctx, deps)
}

func RunReprocess(ctx context.Context, env Env, documentoID string) error {
	catalog, err := openCatalog(ctx, env)
	if err != nil {
		return err
	}
	defer catalog.Close()
	deps, err := buildRunDeps(ctx, env, catalog)
	if err != nil {
		return err
	}
	return application.ReprocessDocumento(ctx, deps, domain.DocumentoID(documentoID))
}

func buildRunDeps(ctx context.Context, env Env, catalog *store.Catalog) (application.RunDailyJobDeps, error) {
	var ext domain.Extrator
	if env.UseStubExtrator {
		log.Printf("using Extrator stub (EXTRATOR_STUB=1 or no Extrator API key)")
		ext = extrator.Stub{Candidatos: nil}
	} else {
		// EXTRATOR_PROVIDER=cursor|gemini|auto (empty=auto). Auto = Gemini first, Cursor on rate-limit (ADR 0037).
		loc, err := time.LoadLocation("America/Sao_Paulo")
		if err != nil {
			return application.RunDailyJobDeps{}, err
		}
		cota := store.NewCotaRepo(catalog)
		switch env.ExtratorProvider {
		case "cursor":
			c, err := extrator.NewCursor(extrator.CursorConfig{
				APIKey:             env.CursorAPIKey,
				Model:              env.CursorModel,
				PythonPath:         env.CursorPython,
				PromptPath:         env.ExtratorPromptPath,
				ExtracaoSchemaPath: env.ExtracaoSchemaPath,
				OfertaSchemaPath:   env.OfertaSchemaPath,
				Logger:             log.Default(),
			})
			if err != nil {
				return application.RunDailyJobDeps{}, fmt.Errorf("cursor extrator: %w", err)
			}
			model := env.CursorModel
			if model == "" {
				model = "composer-2.5"
			}
			log.Printf("using Cursor Extrator model=%s", model)
			ext = c
		case "gemini":
			g, err := extrator.NewGemini(ctx, extrator.GeminiConfig{
				APIKey:             env.GeminiAPIKey,
				Model:              env.GeminiModel,
				PromptPath:         env.ExtratorPromptPath,
				ExtracaoSchemaPath: env.ExtracaoSchemaPath,
				OfertaSchemaPath:   env.OfertaSchemaPath,
				Logger:             log.Default(),
			})
			if err != nil {
				return application.RunDailyJobDeps{}, fmt.Errorf("gemini extrator: %w", err)
			}
			model := env.GeminiModel
			if model == "" {
				model = "gemini-3-flash-preview"
			}
			log.Printf("using Gemini Extrator model=%s", model)
			ext = g
		default: // auto
			var primary, secondary domain.Extrator
			if env.GeminiAPIKey != "" {
				g, err := extrator.NewGemini(ctx, extrator.GeminiConfig{
					APIKey:             env.GeminiAPIKey,
					Model:              env.GeminiModel,
					PromptPath:         env.ExtratorPromptPath,
					ExtracaoSchemaPath: env.ExtracaoSchemaPath,
					OfertaSchemaPath:   env.OfertaSchemaPath,
					Logger:             log.Default(),
				})
				if err != nil {
					return application.RunDailyJobDeps{}, fmt.Errorf("gemini extrator: %w", err)
				}
				primary = g
				model := env.GeminiModel
				if model == "" {
					model = "gemini-3-flash-preview"
				}
				log.Printf("using Gemini Extrator (primary) model=%s", model)
			}
			if env.CursorAPIKey != "" {
				c, err := extrator.NewCursor(extrator.CursorConfig{
					APIKey:             env.CursorAPIKey,
					Model:              env.CursorModel,
					PythonPath:         env.CursorPython,
					PromptPath:         env.ExtratorPromptPath,
					ExtracaoSchemaPath: env.ExtracaoSchemaPath,
					OfertaSchemaPath:   env.OfertaSchemaPath,
					Logger:             log.Default(),
				})
				if err != nil {
					return application.RunDailyJobDeps{}, fmt.Errorf("cursor extrator: %w", err)
				}
				secondary = c
				model := env.CursorModel
				if model == "" {
					model = "composer-2.5"
				}
				log.Printf("using Cursor Extrator (failover) model=%s", model)
			}
			if primary != nil && secondary != nil {
				ext = &extrator.Failover{
					Primary:           primary,
					Secondary:         secondary,
					PrimaryProvider:   domain.ExtratorProviderGemini,
					SecondaryProvider: domain.ExtratorProviderCursor,
					Cota:              cota,
					Location:          loc,
					Log:               log.Default(),
				}
			} else if primary != nil {
				ext = primary
			} else if secondary != nil {
				ext = secondary
			} else {
				return application.RunDailyJobDeps{}, fmt.Errorf("no Extrator API key configured")
			}
		}
	}

	return application.RunDailyJobDeps{
		Fontes:              store.NewFonteRepo(catalog),
		Documentos:          store.NewDocumentoRepo(catalog),
		Produtos:            store.NewProdutoRepo(catalog),
		Marcas:              store.NewMarcaRepo(catalog),
		Ofertas:             store.NewOfertaRepo(catalog),
		Falhas:              store.NewFalhaRepo(catalog),
		Usos:                store.NewUsoExtratorRepo(catalog),
		FonteHTTP:           fontehttp.New(http.DefaultClient),
		Raster:              raster.NewPdftoppm(env.RasterMaxPx, env.RasterJPEGQ),
		Extrator:            ext,
		Artefatos:           artefato.NewLocalStore(env.ArtefatoRoot),
		Dates:               filenamedate.Parser{},
		Log:                 log.Default(),
		ExtratorRetryBudget: time.Hour,
		OnlyFonteID:         domain.FonteID(env.RunFonteID),
		MaxDocumentos:       env.RunMaxDocumentos,
	}, nil
}
