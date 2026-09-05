package domain

import (
	"context"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

// NormalizarRotulo collapses whitespace and lowercases for match-or-create keys.
func NormalizarRotulo(s string) string {
	return store.NormalizarRotulo(s)
}

// MatchOrCreateMarca returns the Marca for nomeNorm, creating it with newID when missing.
func MatchOrCreateMarca(ctx context.Context, repo MarcaRepository, nome string, newID MarcaID) (Marca, error) {
	return store.MatchOrCreateMarca(ctx, repo, nome, newID)
}

// UnirCategorias merges existing catalog categorias with newly seen ones (ADR 0012).
func UnirCategorias(existentes, novas []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(existentes)+len(novas))
	for _, list := range [][]string{existentes, novas} {
		for _, raw := range list {
			n := NormalizarRotulo(raw)
			if n == "" {
				continue
			}
			if _, ok := seen[n]; ok {
				continue
			}
			seen[n] = struct{}{}
			out = append(out, n)
		}
	}
	return out
}
