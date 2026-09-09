package domain

import (
	"strings"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

// ProdutoDoCartao is the sellable type on the card: label without Marca.
func ProdutoDoCartao(nome, marca string) string {
	fields := strings.Fields(strings.TrimSpace(nome))
	if len(fields) == 0 {
		return ""
	}
	brand := store.NormalizarRotulo(marca)
	if brand == "" {
		return strings.Join(fields, " ")
	}
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if store.NormalizarRotulo(f) == brand {
			continue
		}
		out = append(out, f)
	}
	if len(out) == 0 {
		return strings.Join(fields, " ")
	}
	return strings.Join(out, " ")
}

// OfertaDaColeta builds a catalog Oferta from a complete vitrine card.
func OfertaDaColeta(id store.OfertaID, produtoID store.ProdutoID, mercadoID store.MercadoID, marcaID *store.MarcaID, dia string, c CartaoVitrine) store.Oferta {
	return store.Oferta{
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
}
