package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
)

type Config struct {
	DatabaseURL   string
	HTTPAddr      string
	GatewayToken  string
	Allowlist     string
	AckTexto      string
	StubCanal     bool
	MidiaRoot     string
	CORSOrigin    string
	AgenteURL     string
	WhatsAppToken string
	PhoneNumberID string
	DisplayNumber string
	VerifyToken   string
	AppSecret     string
	GraphVersion  string
}

func Load() (Config, error) {
	loadDotEnv(findDotEnv())
	cfg := Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		HTTPAddr:      envOr("HTTP_ADDR", ":8090"),
		GatewayToken:  os.Getenv("GATEWAY_TOKEN"),
		Allowlist:     os.Getenv("WHATSAPP_ALLOWLIST"),
		AckTexto:      envOr("WHATSAPP_ACK_TEXTO", "Recebi. O ajudante de ofertas ainda não está ligado neste canal."),
		StubCanal:     truthy(os.Getenv("WHATSAPP_STUB")),
		MidiaRoot:     envOr("MIDIA_ROOT", "./.data/midia"),
		CORSOrigin:    envOr("CORS_ORIGIN", "http://localhost:5173"),
		AgenteURL:     strings.TrimSpace(os.Getenv("AGENTE_URL")),
		WhatsAppToken: strings.TrimSpace(os.Getenv("WHATSAPP_TOKEN")),
		PhoneNumberID: strings.TrimSpace(os.Getenv("WHATSAPP_PHONE_NUMBER_ID")),
		DisplayNumber: strings.TrimSpace(os.Getenv("WHATSAPP_DISPLAY_NUMBER")),
		VerifyToken:   strings.TrimSpace(os.Getenv("WHATSAPP_VERIFY_TOKEN")),
		AppSecret:     strings.TrimSpace(os.Getenv("WHATSAPP_APP_SECRET")),
		GraphVersion:  envOr("WHATSAPP_GRAPH_VERSION", "v21.0"),
	}
	var err error
	cfg.MidiaRoot, err = filepath.Abs(cfg.MidiaRoot)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) DisplayJID() domain.JID {
	if c.DisplayNumber == "" {
		return ""
	}
	return domain.NormalizarJID(c.DisplayNumber)
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
