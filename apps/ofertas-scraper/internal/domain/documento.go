package domain

type EstadoDocumento string

const (
	EstadoProcessando EstadoDocumento = "processando"
	EstadoConcluido   EstadoDocumento = "concluido"
	EstadoParcial     EstadoDocumento = "parcial"
	EstadoFalhou      EstadoDocumento = "falhou"
)

// EstadoAposValidacao decides the Documento terminal state after extraction validation.
// Zero persisted Ofertas → falhou (ADR 0006).
func EstadoAposValidacao(ofertasValidas, falhas int) EstadoDocumento {
	if ofertasValidas == 0 {
		return EstadoFalhou
	}
	if falhas > 0 {
		return EstadoParcial
	}
	return EstadoConcluido
}

// DeveReprocessar reports whether a same-day re-run should process this Documento.
func DeveReprocessar(estado EstadoDocumento) bool {
	switch estado {
	case EstadoConcluido, EstadoParcial:
		return false
	default:
		return true
	}
}
