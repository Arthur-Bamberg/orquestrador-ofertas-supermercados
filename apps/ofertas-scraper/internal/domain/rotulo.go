package domain

import "strings"

// NormalizarRotulo collapses whitespace and lowercases for match-or-create keys.
func NormalizarRotulo(s string) string {
	fields := strings.Fields(strings.TrimSpace(s))
	if len(fields) == 0 {
		return ""
	}
	return strings.ToLower(strings.Join(fields, " "))
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
