package application

import "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"

// CandidatoResolvido is a candidate after vigência cascades, with date origens (ADR 0028).
type CandidatoResolvido struct {
	Candidato           domain.CandidatoOferta
	OrigemDataInicio    domain.OrigemData
	OrigemDataExpiracao domain.OrigemData
}

// ResolverVigencia fills missing dataInicio/dataExpiracao and records origens.
// diaPrimeiraDescoberta must be the earliest discovery day for Fonte+filename (or the current dia).
func ResolverVigencia(
	c domain.CandidatoOferta,
	filename string,
	parser domain.FilenameDateParser,
	diaPrimeiraDescoberta string,
) CandidatoResolvido {
	var file domain.FilenameVigencia
	if parser != nil {
		file = parser.Parse(filename)
	}

	out := c
	var origemInicio, origemFim domain.OrigemData

	if out.DataInicio != "" {
		origemInicio = domain.OrigemExtrator
	} else if file.DataInicio != "" {
		out.DataInicio = file.DataInicio
		origemInicio = domain.OrigemFilename
	} else {
		out.DataInicio = diaPrimeiraDescoberta
		origemInicio = domain.OrigemPrimeiraDescoberta
	}

	if out.DataExpiracao != "" {
		origemFim = domain.OrigemExtrator
	} else if file.DataExpiracao != "" {
		out.DataExpiracao = file.DataExpiracao
		origemFim = domain.OrigemFilename
	}
	// empty fim → Falha in domain validation

	return CandidatoResolvido{
		Candidato:           out,
		OrigemDataInicio:    origemInicio,
		OrigemDataExpiracao: origemFim,
	}
}
