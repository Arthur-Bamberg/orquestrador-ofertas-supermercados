package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_leDotEnvDoDiretorioPai(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "apps", "gateway-whatsapp")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	unset(t, "DATABASE_URL")
	unset(t, "GATEWAY_TOKEN")
	unset(t, "WHATSAPP_ALLOWLIST")
	unset(t, "HTTP_ADDR")
	unset(t, "WHATSAPP_STUB")
	unset(t, "WHATSAPP_ACK_TEXTO")
	unset(t, "WHATSAPP_SESSION_PATH")
	unset(t, "MIDIA_ROOT")
	body := "DATABASE_URL=postgres://from-root/ofertas\nGATEWAY_TOKEN=from-root\nWHATSAPP_ALLOWLIST=5511999999999\nWHATSAPP_STUB=1\n"
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(child)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgres://from-root/ofertas" || cfg.GatewayToken != "from-root" {
		t.Fatalf("%+v", cfg)
	}
}

func TestLoad_defaultsEAllowlist(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("DATABASE_URL", "postgres://ofertas:ofertas@localhost:5432/ofertas?sslmode=disable")
	t.Setenv("GATEWAY_TOKEN", "secret")
	t.Setenv("WHATSAPP_ALLOWLIST", "5511999999999")
	t.Setenv("WHATSAPP_STUB", "1")
	unset(t, "HTTP_ADDR")
	unset(t, "WHATSAPP_ACK_TEXTO")
	unset(t, "WHATSAPP_SESSION_PATH")
	unset(t, "MIDIA_ROOT")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8090" || !cfg.StubCanal || cfg.Allowlist != "5511999999999" {
		t.Fatalf("%+v", cfg)
	}
	if cfg.AckTexto == "" {
		t.Fatal("ack default")
	}
	if !filepath.IsAbs(cfg.SessionPath) || !filepath.IsAbs(cfg.MidiaRoot) {
		t.Fatalf("paths %+v", cfg)
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
