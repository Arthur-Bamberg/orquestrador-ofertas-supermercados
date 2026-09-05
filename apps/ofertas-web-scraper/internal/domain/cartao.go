package domain

import (
	"strings"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

// NormalizarRotulo collapses whitespace and lowercases for catalog matching.
func NormalizarRotulo(s string) string {
	return store.NormalizarRotulo(s)
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
