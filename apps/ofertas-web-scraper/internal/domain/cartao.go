package domain

import "strings"

// NormalizarRotulo collapses whitespace and lowercases for catalog matching.
func NormalizarRotulo(s string) string {
	fields := strings.Fields(strings.TrimSpace(s))
	if len(fields) == 0 {
		return ""
	}
	return strings.ToLower(strings.Join(fields, " "))
}

// CartaoCasaProduto reports whether a vitrine card is the searched sellable type.
// The Produto nomeNorm must appear in the card label; brand and size may vary.
func CartaoCasaProduto(nomeCartao, produtoNomeNorm string) bool {
	cartao := NormalizarRotulo(nomeCartao)
	produto := NormalizarRotulo(produtoNomeNorm)
	if cartao == "" || produto == "" {
		return false
	}
	return strings.Contains(cartao, produto)
}
