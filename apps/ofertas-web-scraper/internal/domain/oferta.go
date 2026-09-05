package domain

import store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"

// OfertaDaColeta builds a catalog Oferta from a matching vitrine card.
func OfertaDaColeta(id store.OfertaID, produto store.Produto, mercadoID store.MercadoID, marcaID *store.MarcaID, dia string, c CartaoVitrine) store.Oferta {
	return store.Oferta{
		ID:                   id,
		ProdutoID:            produto.ID,
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

// CartaoCompleto reports whether the card has the fields required to persist an Oferta.
func CartaoCompleto(c CartaoVitrine) bool {
	if c.Valor <= 0 || len(c.Quantidades) == 0 {
		return false
	}
	switch c.Medida {
	case store.MedidaG, store.MedidaML, store.MedidaUnidade:
		return true
	default:
		return false
	}
}
