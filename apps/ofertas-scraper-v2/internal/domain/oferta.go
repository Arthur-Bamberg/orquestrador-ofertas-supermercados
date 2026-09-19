package domain

import (
	"regexp"
	"strings"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

// Global regex for size patterns (compile once)
var sizePattern = regexp.MustCompile(`(?i)\b(?:caixa|lata|pouch|sachê|pct|pacote)\s*(?:[[:digit:]]+(?:[.,][[:digit:]]+)?\s*(?:kg|g|l|ml|un|cx|caixa|pct|pacote|lt|litro|x|und?))?\b|\b[[:digit:]]+(?:[.,][[:digit:]]+)?\s*(?:kg|g|l|ml|un|cx|caixa|pct|pacote|lt|litro|x|und?|lata|pouch|sachê)\b`)

// Regex for common tokens that indicate size/quantity in a product name but aren't units (e.g. "caixa", "pct")
var sizeTokenPattern = regexp.MustCompile(`(?i)\b(?:caixa|lata|pouch|sachê|pct|pacote|bandeja|display)\s*\b`)


// ProdutoDoCartao is the sellable type on the card: label without Marca.
func ProdutoDoCartao(nome, marca string) string {
	cleanedName := strings.TrimSpace(nome)

		// 1. Remove brand from name if present
	brandLower := strings.ToLower(marca)
	if brandLower != "" && strings.Contains(strings.ToLower(cleanedName), brandLower) {
		cleanedName = strings.ReplaceAll(strings.ToLower(cleanedName), brandLower, "")
	}

	// 2. Remove size patterns and common size tokens
	cleanedName = sizePattern.ReplaceAllString(cleanedName, "")
	cleanedName = sizeTokenPattern.ReplaceAllString(cleanedName, "")

	// Clean up multiple spaces that might result from removals and trim final string
	cleanedName = regexp.MustCompile(`\s+`).ReplaceAllString(cleanedName, " ")
	cleanedName = strings.TrimSpace(cleanedName)

	return cleanedName
}

// OfertaDaColeta builds a catalog Oferta from a complete vitrine card.
func OfertaDaColeta(id store.OfertaID, produtoID store.ProdutoID, mercadoID store.MercadoID, marcaID *store.MarcaID, dia string, c CartaoVitrine) store.Oferta {
	oferta := store.Oferta{
		ID:                   id,
		ProdutoID:            produtoID,
		MarcaID:              marcaID,
		MercadoID:            mercadoID,
		Valor:                c.Valor,
		Quantidades:          c.Quantidades,
		Medida:               c.Medida,
		DataInicio:           dia,
		DataExpiracao:        dia,
		OrigemDataInicio:     store.OrigemColeta,
		OrigemDataExpiracao:  store.OrigemColeta,
		IndicacaoPromocional: c.IndicacaoPromocional,
	}

	// If the card measure is 'unidade' or it's a generic weight/volume, try to extract quantity and measure from the product name
	// This helps in cases like "Coxa e Sobrecoxa Congelada 1kg" where the card might be "unidade" but the product has a weight
	if c.Medida == store.MedidaUnidade || (c.Medida == store.MedidaG && c.Quantidades[0] == 1) || (c.Medida == store.MedidaML && c.Quantidades[0] == 1) {
		if q, m, ok := store.ExtractQuantityAndMeasure(c.Nome); ok {
			// Only update if extracted quantity/measure is more specific than current
			if (oferta.Medida == store.MedidaUnidade) || (oferta.Medida == store.MedidaG && oferta.Quantidades[0] == 1) || (oferta.Medida == store.MedidaML && oferta.Quantidades[0] == 1) {
				oferta.Quantidades = []float64{q}
				oferta.Medida = m
			}
		}
	}

	return oferta
}
