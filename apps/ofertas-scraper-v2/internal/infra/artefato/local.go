package artefato

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
)

// Store writes page JPEGs under {root}/{mercado}/{encarteID}/pagina-NN.jpeg.
type Store struct {
	Root string
}

func New(root string) *Store {
	return &Store{Root: root}
}

func (s *Store) Save(mercado, encarteID string, pagina domain.Pagina, body []byte) (string, error) {
	if s == nil || s.Root == "" {
		return "", fmt.Errorf("artefato root vazio")
	}
	rel := filepath.Join(slug(mercado), encarteID, fmt.Sprintf("pagina-%02d.jpeg", pagina.Numero))
	abs := filepath.Join(s.Root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, body, 0o644); err != nil {
		return "", err
	}
	return rel, nil
}

func slug(nome string) string {
	n := domain.NormalizarNome(nome)
	var b strings.Builder
	dash := true
	for _, r := range n {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			dash = false
		default:
			if !dash {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return "mercado"
	}
	return s
}
