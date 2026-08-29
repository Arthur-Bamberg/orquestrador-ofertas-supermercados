package domain

import store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"

type EstadoDocumento = store.EstadoDocumento

const (
	EstadoProcessando = store.EstadoProcessando
	EstadoConcluido   = store.EstadoConcluido
	EstadoParcial     = store.EstadoParcial
	EstadoFalhou      = store.EstadoFalhou
)

// EstadoAposValidacao decides the Documento terminal state after extraction validation.
// Zero persisted Ofertas → falhou (ADR 0006).
func EstadoAposValidacao(ofertasValidas, falhas int) EstadoDocumento {
	return store.EstadoAposValidacao(ofertasValidas, falhas)
}

// DeveReprocessar reports whether a same-day re-run should process this Documento.
func DeveReprocessar(estado EstadoDocumento) bool {
	return store.DeveReprocessar(estado)
}
