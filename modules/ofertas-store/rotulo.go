package store

import (
	"context"
	"strings"
)

// NormalizarRotulo collapses whitespace and lowercases for match-or-create keys.
func NormalizarRotulo(s string) string {
	fields := strings.Fields(strings.TrimSpace(s))
	if len(fields) == 0 {
		return ""
	}
	return strings.ToLower(strings.Join(fields, " "))
}

type marcaLookup interface {
	GetByNomeNorm(ctx context.Context, nomeNorm string) (Marca, bool, error)
	Save(ctx context.Context, m Marca) error
}

// MatchOrCreateMarca returns the Marca for nomeNorm, creating it with newID when missing.
func MatchOrCreateMarca(ctx context.Context, repo marcaLookup, nome string, newID MarcaID) (Marca, error) {
	norm := NormalizarRotulo(nome)
	existing, ok, err := repo.GetByNomeNorm(ctx, norm)
	if err != nil {
		return Marca{}, err
	}
	if ok {
		return existing, nil
	}
	m := Marca{ID: newID, Nome: nome, NomeNorm: norm}
	if err := repo.Save(ctx, m); err != nil {
		return Marca{}, err
	}
	return m, nil
}
