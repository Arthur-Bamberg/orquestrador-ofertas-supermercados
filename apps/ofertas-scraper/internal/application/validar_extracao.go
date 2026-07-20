package application

import "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"

// ValidarExtracao validates candidates that already carry dates (assumes origem extrator).
func ValidarExtracao(candidatos []domain.CandidatoOferta) (validas []domain.OfertaValidada, falhas []domain.FalhaExtracao, estado domain.EstadoDocumento) {
	resolvidos := make([]CandidatoResolvido, len(candidatos))
	for i, c := range candidatos {
		r := CandidatoResolvido{Candidato: c}
		if c.DataInicio != "" {
			r.OrigemDataInicio = domain.OrigemExtrator
		}
		if c.DataExpiracao != "" {
			r.OrigemDataExpiracao = domain.OrigemExtrator
		}
		resolvidos[i] = r
	}
	return ValidarExtracaoResolvida(resolvidos)
}

// ValidarExtracaoResolvida validates candidates that already have vigência/origens resolved.
func ValidarExtracaoResolvida(resolvidos []CandidatoResolvido) (validas []domain.OfertaValidada, falhas []domain.FalhaExtracao, estado domain.EstadoDocumento) {
	for _, r := range resolvidos {
		oferta, falha := domain.ValidarCandidato(r.Candidato)
		if falha != nil {
			falhas = append(falhas, *falha)
			continue
		}
		oferta.OrigemDataInicio = r.OrigemDataInicio
		oferta.OrigemDataExpiracao = r.OrigemDataExpiracao
		validas = append(validas, oferta)
	}
	estado = domain.EstadoAposValidacao(len(validas), len(falhas))
	return validas, falhas, estado
}
