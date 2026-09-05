package domain

import "strings"

// FiltrarEncartes keeps Encartes whose Mercado matches a name we already have.
func FiltrarEncartes(encartes []Encarte, mercados []string) []Encarte {
	want := map[string]struct{}{}
	for _, nome := range mercados {
		n := NormalizarNome(nome)
		if n == "" {
			continue
		}
		want[n] = struct{}{}
	}
	if len(want) == 0 {
		return nil
	}
	var out []Encarte
	for _, e := range encartes {
		if _, ok := want[NormalizarNome(e.Mercado)]; ok {
			out = append(out, e)
		}
	}
	return out
}

// NormalizarNome collapses whitespace and lowercases a Mercado label.
func NormalizarNome(s string) string {
	fields := strings.Fields(strings.TrimSpace(s))
	if len(fields) == 0 {
		return ""
	}
	return strings.ToLower(strings.Join(fields, " "))
}
