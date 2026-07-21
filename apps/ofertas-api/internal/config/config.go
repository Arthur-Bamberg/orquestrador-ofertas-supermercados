package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	UpstashURL   string
	UpstashToken string
	HTTPAddr     string
	CORSOrigin   string
	ScraperDir   string
	ArtefatoRoot string
}

func Load() (Config, error) {
	cfg := Config{
		UpstashURL:   os.Getenv("UPSTASH_REDIS_REST_URL"),
		UpstashToken: os.Getenv("UPSTASH_REDIS_REST_TOKEN"),
		HTTPAddr:     envOr("HTTP_ADDR", ":8080"),
		CORSOrigin:   envOr("CORS_ORIGIN", "http://localhost:5173"),
		ScraperDir:   envOr("SCRAPER_DIR", "../ofertas-scraper"),
		ArtefatoRoot: envOr("ARTEFATO_ROOT", "../ofertas-scraper/.data/artefatos"),
	}
	var err error
	cfg.ScraperDir, err = filepath.Abs(cfg.ScraperDir)
	if err != nil {
		return Config{}, err
	}
	cfg.ArtefatoRoot, err = filepath.Abs(cfg.ArtefatoRoot)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
