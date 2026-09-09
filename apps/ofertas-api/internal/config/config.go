package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	DatabaseURL  string
	HTTPAddr     string
	CORSOrigin   string
	ScraperDir   string
	ScraperBin   string
	ArtefatoRoot string
	ColetaURL    string
	ColetaToken  string
}

func Load() (Config, error) {
	loadDotEnv(".env")
	cfg := Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		HTTPAddr:     envOr("HTTP_ADDR", ":8080"),
		CORSOrigin:   envOr("CORS_ORIGIN", "http://localhost:5173"),
		ScraperDir:   envOr("SCRAPER_DIR", "../ofertas-scraper"),
		ScraperBin:   os.Getenv("SCRAPER_BIN"),
		ArtefatoRoot: envOr("ARTEFATO_ROOT", "../ofertas-scraper/.data/artefatos"),
		ColetaURL:    strings.TrimRight(os.Getenv("COLETA_URL"), "/"),
		ColetaToken:  os.Getenv("GATEWAY_TOKEN"),
	}
	var err error
	cfg.ScraperDir, err = filepath.Abs(cfg.ScraperDir)
	if err != nil {
		return Config{}, err
	}
	if cfg.ScraperBin != "" {
		cfg.ScraperBin, err = filepath.Abs(cfg.ScraperBin)
		if err != nil {
			return Config{}, err
		}
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

// loadDotEnv sets KEY=VALUE from a local .env if the key is not already in the environment.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		_ = os.Setenv(key, value)
	}
}
