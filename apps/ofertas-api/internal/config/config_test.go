package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_EnvAndDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("DATABASE_URL", "postgres://ofertas:ofertas@localhost:5432/ofertas?sslmode=disable")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("CORS_ORIGIN", "http://front.test")
	t.Setenv("SCRAPER_DIR", dir)
	t.Setenv("SCRAPER_BIN", filepath.Join(dir, "ofertas-scraper"))
	t.Setenv("ARTEFATO_ROOT", filepath.Join(dir, "art"))

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgres://ofertas:ofertas@localhost:5432/ofertas?sslmode=disable" {
		t.Fatalf("database %+v", cfg)
	}
	if cfg.HTTPAddr != ":9090" || cfg.CORSOrigin != "http://front.test" {
		t.Fatalf("http %+v", cfg)
	}
	if cfg.ScraperDir != dir {
		t.Fatalf("scraperDir=%s want %s", cfg.ScraperDir, dir)
	}
	if cfg.ScraperBin != filepath.Join(dir, "ofertas-scraper") {
		t.Fatalf("scraperBin=%s", cfg.ScraperBin)
	}
	if cfg.ArtefatoRoot != filepath.Join(dir, "art") {
		t.Fatalf("artefatoRoot=%s", cfg.ArtefatoRoot)
	}
}

func TestLoad_DotEnvWhenKeyUnset(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	unset(t, "DATABASE_URL")
	unset(t, "HTTP_ADDR")
	if err := os.WriteFile(".env", []byte("DATABASE_URL=postgres://from-env/ofertas\nHTTP_ADDR=:7070\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgres://from-env/ofertas" {
		t.Fatalf("url=%s", cfg.DatabaseURL)
	}
	if cfg.HTTPAddr != ":7070" {
		t.Fatalf("addr=%s", cfg.HTTPAddr)
	}
}

func unset(t *testing.T, key string) {
	t.Helper()
	old, had := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}
