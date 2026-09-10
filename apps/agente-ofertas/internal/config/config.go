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
	GatewayURL   string
	GatewayToken string
	ColetaURL    string
}

func Load() (Config, error) {
	loadDotEnv(findDotEnv())
	return Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		HTTPAddr:     envOr("HTTP_ADDR", ":8091"),
		GatewayURL:   envOr("GATEWAY_URL", "http://localhost:8090"),
		GatewayToken: os.Getenv("GATEWAY_TOKEN"),
		ColetaURL:    strings.TrimRight(os.Getenv("COLETA_URL"), "/"),
	}, nil
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
