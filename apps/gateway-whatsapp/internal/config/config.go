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
	GatewayToken string
	Allowlist    string
	AckTexto     string
	AutoNome     string
	AutoTexto    string
	StubCanal    bool
	SessionPath  string
	MidiaRoot    string
	CORSOrigin   string
}

func Load() (Config, error) {
	loadDotEnv(findDotEnv())
	cfg := Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		HTTPAddr:     envOr("HTTP_ADDR", ":8090"),
		GatewayToken: os.Getenv("GATEWAY_TOKEN"),
		Allowlist:    os.Getenv("WHATSAPP_ALLOWLIST"),
		AckTexto:     envOr("WHATSAPP_ACK_TEXTO", "Recebi. O ajudante de ofertas ainda não está ligado neste canal."),
		AutoNome:     os.Getenv("WHATSAPP_AUTO_NOME"),
		AutoTexto:    os.Getenv("WHATSAPP_AUTO_TEXTO"),
		StubCanal:    truthy(os.Getenv("WHATSAPP_STUB")),
		SessionPath:  envOr("WHATSAPP_SESSION_PATH", "./.data/whatsmeow.db"),
		MidiaRoot:    envOr("MIDIA_ROOT", "./.data/midia"),
		CORSOrigin:   envOr("CORS_ORIGIN", "http://localhost:5173"),
	}
	var err error
	cfg.SessionPath, err = filepath.Abs(cfg.SessionPath)
	if err != nil {
		return Config{}, err
	}
	cfg.MidiaRoot, err = filepath.Abs(cfg.MidiaRoot)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func findDotEnv() string {
	dir, err := os.Getwd()
	if err != nil {
		return ".env"
	}
	for {
		p := filepath.Join(dir, ".env")
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ".env"
		}
		dir = parent
	}
}

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
