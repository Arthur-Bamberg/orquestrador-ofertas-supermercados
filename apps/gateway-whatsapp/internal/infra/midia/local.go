package midia

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
)

type Local struct {
	Root string
}

func (l Local) Guardar(_ context.Context, mensagemID domain.MensagemID, midia domain.MidiaBytes) (string, error) {
	if err := os.MkdirAll(l.Root, 0o755); err != nil {
		return "", err
	}
	name := string(mensagemID)
	if ext := filepath.Ext(midia.Filename); ext != "" {
		name += ext
	}
	full, err := l.resolve(name)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(full, midia.Conteudo, 0o644); err != nil {
		return "", err
	}
	return name, nil
}

func (l Local) Ler(_ context.Context, path string) ([]byte, error) {
	full, err := l.resolve(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(full)
}

func (l Local) resolve(path string) (string, error) {
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return "", err
	}
	full := filepath.Join(root, filepath.Clean(path))
	rel, err := filepath.Rel(root, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path de mídia inválido")
	}
	return full, nil
}
