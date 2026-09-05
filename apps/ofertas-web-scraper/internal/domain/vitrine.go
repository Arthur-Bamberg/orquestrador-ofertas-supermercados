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

type Vitrine interface {
	Buscar(ctx context.Context, query string) ([]CartaoVitrine, error)
}
