package application_test

import (
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

type stubDateParser struct {
	v domain.FilenameVigencia
}

func (s stubDateParser) Parse(string) domain.FilenameVigencia { return s.v }

func TestValidarExtracao_listaVaziaEhFalhou(t *testing.T) {
	_, falhas, estado := application.ValidarExtracao(nil)
	if len(falhas) != 0 {
		t.Fatalf("falhas: %d", len(falhas))
	}
	if estado != domain.EstadoFalhou {
		t.Fatalf("estado: got %q", estado)
	}
}

func TestValidarExtracao_mistoParcial(t *testing.T) {
	candidatos := []domain.CandidatoOferta{
		{Produto: "Arroz", Valor: 10, Quantidades: []float64{1000}, Medida: "g", DataInicio: "2026-07-18", DataExpiracao: "2026-07-20"},
		{Produto: "Feijão", Valor: 8, Quantidades: []float64{1}, Medida: "kg", DataInicio: "2026-07-18", DataExpiracao: "2026-07-20"},
	}
	validas, falhas, estado := application.ValidarExtracao(candidatos)
	if len(validas) != 1 || len(falhas) != 1 {
		t.Fatalf("validas=%d falhas=%d", len(validas), len(falhas))
	}
	if estado != domain.EstadoParcial {
		t.Fatalf("estado: got %q", estado)
	}
}

func TestResolverVigencia_cascata(t *testing.T) {
	parser := stubDateParser{v: domain.FilenameVigencia{
		DataInicio: "2026-07-18", DataExpiracao: "2026-07-19",
	}}

	semDatas := domain.CandidatoOferta{Produto: "X", Valor: 1, Quantidades: []float64{1}, Medida: "unidade"}
	got := application.ResolverVigencia(semDatas, "MS_Fort_FDS_18-e-19_JUL_26.pdf", parser, "2026-07-10")
	if got.Candidato.DataInicio != "2026-07-18" || got.OrigemDataInicio != domain.OrigemFilename {
		t.Fatalf("inicio filename: %#v", got)
	}
	if got.Candidato.DataExpiracao != "2026-07-19" || got.OrigemDataExpiracao != domain.OrigemFilename {
		t.Fatalf("fim filename: %#v", got)
	}

	comExtrator := semDatas
	comExtrator.DataInicio = "2026-07-01"
	comExtrator.DataExpiracao = "2026-07-31"
	got = application.ResolverVigencia(comExtrator, "x.pdf", parser, "2026-07-10")
	if got.Candidato.DataInicio != "2026-07-01" || got.OrigemDataInicio != domain.OrigemExtrator {
		t.Fatalf("extrator vence inicio: %#v", got)
	}
	if got.Candidato.DataExpiracao != "2026-07-31" || got.OrigemDataExpiracao != domain.OrigemExtrator {
		t.Fatalf("extrator vence fim: %#v", got)
	}

	soISO := stubDateParser{v: domain.FilenameVigencia{DataExpiracao: "2026-07-25"}}
	got = application.ResolverVigencia(semDatas, "encarte-20260725.pdf", soISO, "2026-07-10")
	if got.Candidato.DataInicio != "2026-07-10" || got.OrigemDataInicio != domain.OrigemPrimeiraDescoberta {
		t.Fatalf("primeira descoberta: %#v", got)
	}
	if got.Candidato.DataExpiracao != "2026-07-25" || got.OrigemDataExpiracao != domain.OrigemFilename {
		t.Fatalf("iso como fim: %#v", got)
	}
}

func TestValidarExtracaoResolvida_preservaOrigens(t *testing.T) {
	resolvidos := []application.CandidatoResolvido{{
		Candidato: domain.CandidatoOferta{
			Produto: "Arroz", Valor: 10, Quantidades: []float64{1000}, Medida: "g",
			DataInicio: "2026-07-18", DataExpiracao: "2026-07-19",
		},
		OrigemDataInicio:    domain.OrigemFilename,
		OrigemDataExpiracao: domain.OrigemFilename,
	}}
	validas, falhas, estado := application.ValidarExtracaoResolvida(resolvidos)
	if len(validas) != 1 || len(falhas) != 0 || estado != domain.EstadoConcluido {
		t.Fatalf("validas=%d falhas=%d estado=%s", len(validas), len(falhas), estado)
	}
	if validas[0].OrigemDataInicio != domain.OrigemFilename || validas[0].OrigemDataExpiracao != domain.OrigemFilename {
		t.Fatalf("origens=%s/%s", validas[0].OrigemDataInicio, validas[0].OrigemDataExpiracao)
	}
}

func TestValidarExtracaoResolvida_fimAusenteEhFalha(t *testing.T) {
	resolvidos := []application.CandidatoResolvido{{
		Candidato: domain.CandidatoOferta{
			Produto: "Arroz", Valor: 10, Quantidades: []float64{1000}, Medida: "g",
			DataInicio: "2026-07-18",
		},
		OrigemDataInicio: domain.OrigemPrimeiraDescoberta,
	}}
	validas, falhas, estado := application.ValidarExtracaoResolvida(resolvidos)
	if len(validas) != 0 || len(falhas) != 1 || estado != domain.EstadoFalhou {
		t.Fatalf("validas=%d falhas=%d estado=%s", len(validas), len(falhas), estado)
	}
	if falhas[0].Codigo != domain.CodigoDataExpiracaoInvalida {
		t.Fatalf("codigo=%s", falhas[0].Codigo)
	}
}
