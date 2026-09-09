package domain

import (
	"context"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

type CartaoVitrine struct {
	Nome                 string
	Marca                string
	Valor                float64
	Quantidades          []float64
	Medida               store.Medida
	IndicacaoPromocional bool
}

// ResultadoVitrine is one Mercado search: cards seen and whether the vitrine was exhausted.
type ResultadoVitrine struct {
	Cartoes  []CartaoVitrine
	Esgotada bool
}

// PaginaVitrine is one page of a vitrine search (hits + reported total / cursor).
type PaginaVitrine struct {
	Cartoes  []CartaoVitrine
	Total    int
	NextFrom int
	HasNext  bool
}

type Vitrine interface {
	Buscar(ctx context.Context, query string) (ResultadoVitrine, error)
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

// EstadoAposColeta maps persisted Ofertas, incomplete-card discards, and whether the search finished.
func EstadoAposColeta(ofertas, descartes int, esgotada bool) store.EstadoDocumento {
	if !esgotada {
		if ofertas == 0 {
			return store.EstadoFalhou
		}
		return store.EstadoParcial
	}
	if ofertas == 0 {
		return store.EstadoConcluido
	}
	if descartes > 0 {
		return store.EstadoParcial
	}
	return store.EstadoConcluido
}

