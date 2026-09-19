package store

import (
	"context"
	"regexp"
	"strconv"
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

// Regex for extracting quantity and unit (e.g., "1kg", "500g", "2L", "200 ml")
var quantityMeasurePattern = regexp.MustCompile(`(?i)([[:digit:]]+(?:[.,][[:digit:]]+)?)\s*(kg|g|l|ml|litro)\b`)

// ExtractQuantityAndMeasure attempts to parse quantity and measure from a product name.
func ExtractQuantityAndMeasure(productName string) (float64, Medida, bool) {
	matches := quantityMeasurePattern.FindStringSubmatch(productName)
	if len(matches) < 3 {
		return 0, "", false
	}
	quantityStr := strings.Replace(matches[1], ",", ".", 1) // Replace comma with dot for float parsing
	quantity, err := strconv.ParseFloat(quantityStr, 64)
	if err != nil {
		return 0, "", false
	}
	unit := strings.ToLower(matches[2])
	var medida Medida
	switch unit {
	case "kg":
		medida = MedidaG
		quantity *= 1000 // Convert kg to g
	case "g":
		medida = MedidaG
	case "l", "litro":
		medida = MedidaML
		quantity *= 1000 // Convert L to ml
	case "ml":
		medida = MedidaML
	default:
		return 0, "", false
	}
	return quantity, medida, true
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

type produtoLookup interface {
	GetByNomeNorm(ctx context.Context, nomeNorm string) (Produto, bool, error)
	Save(ctx context.Context, p Produto) error
}

// MatchOrCreateProduto returns the Produto for nomeNorm, creating it with newID when missing.
func MatchOrCreateProduto(ctx context.Context, repo produtoLookup, nome string, newID ProdutoID) (Produto, error) {
	norm := NormalizarRotulo(nome)
	existing, ok, err := repo.GetByNomeNorm(ctx, norm)
	if err != nil {
		return Produto{}, err
	}
	if ok {
		return existing, nil
	}
	p := Produto{ID: newID, Nome: nome, NomeNorm: norm}
	if err := repo.Save(ctx, p); err != nil {
		return Produto{}, err
	}
	return p, nil
}
