package domain

import (
	"sort"
	"strings"
	"time"
)

const (
	CodigoMedidaInvalida        = "medida_invalida"
	CodigoProdutoInvalido       = "produto_invalido"
	CodigoValorInvalido         = "valor_invalido"
	CodigoQuantidadeInvalida    = "quantidade_invalida"
	CodigoDataInicioInvalida    = "data_inicio_invalida"
	CodigoDataExpiracaoInvalida = "data_expiracao_invalida"
	CodigoVigenciaInvalida      = "vigencia_invalida"
	CodigoPromocaoInvalida      = "promocao_invalida"
	CodigoComparativoInvalido   = "comparativo_invalido"
)

// OfertaValidada is a candidate that passed domain validation (labels, not ids).
type OfertaValidada struct {
	Produto             string
	Marca               string
	Categorias          []string
	Valor               float64
	Quantidades         []float64
	Medida              Medida
	DataInicio          string
	DataExpiracao       string
	OrigemDataInicio    OrigemData
	OrigemDataExpiracao OrigemData
	Promocao            *Promocao
	Comparativo         *Comparativo
}

func ValidarCandidato(c CandidatoOferta) (OfertaValidada, *FalhaExtracao) {
	produto := strings.TrimSpace(c.Produto)
	if produto == "" {
		return falha(c, CodigoProdutoInvalido, "produto é obrigatório")
	}

	if c.Valor <= 0 {
		return falha(c, CodigoValorInvalido, "valor deve ser > 0")
	}

	quantidades, ok := normalizarQuantidades(c.Quantidades)
	if !ok {
		return falha(c, CodigoQuantidadeInvalida, "quantidades deve ter ≥1 valor >= 0")
	}

	medida, ok := parseMedida(c.Medida)
	if !ok {
		return falha(c, CodigoMedidaInvalida, "medida deve ser g, ml ou unidade")
	}

	inicio, err := time.Parse("2006-01-02", c.DataInicio)
	if err != nil {
		return falha(c, CodigoDataInicioInvalida, "dataInicio deve ser YYYY-MM-DD")
	}
	fim, err := time.Parse("2006-01-02", c.DataExpiracao)
	if err != nil {
		return falha(c, CodigoDataExpiracaoInvalida, "dataExpiracao deve ser YYYY-MM-DD")
	}
	if inicio.After(fim) {
		return falha(c, CodigoVigenciaInvalida, "dataInicio deve ser ≤ dataExpiracao")
	}

	if falhaPromo := validarPromocao(c); falhaPromo != nil {
		return OfertaValidada{}, falhaPromo
	}
	if falhaComp := validarComparativo(c, quantidades); falhaComp != nil {
		return OfertaValidada{}, falhaComp
	}

	return OfertaValidada{
		Produto:       produto,
		Marca:         strings.TrimSpace(c.Marca),
		Categorias:    normalizarCategorias(c.Categorias),
		Valor:         c.Valor,
		Quantidades:   quantidades,
		Medida:        medida,
		DataInicio:    c.DataInicio,
		DataExpiracao: c.DataExpiracao,
		Promocao:      c.Promocao,
		Comparativo:   c.Comparativo,
	}, nil
}

func falha(c CandidatoOferta, codigo, detalhe string) (OfertaValidada, *FalhaExtracao) {
	return OfertaValidada{}, &FalhaExtracao{
		Codigo:    codigo,
		Detalhe:   detalhe,
		Candidato: c,
	}
}

func parseMedida(s string) (Medida, bool) {
	switch Medida(s) {
	case MedidaG, MedidaML, MedidaUnidade:
		return Medida(s), true
	default:
		return "", false
	}
}

func coalesceQuantidades(quantidades []float64, legacy *float64) []float64 {
	if len(quantidades) > 0 {
		return quantidades
	}
	if legacy != nil {
		return []float64{*legacy}
	}
	return nil
}

// normalizarQuantidades dedups and sorts ascending (ADR 0030). Rejects empty or negative.
func normalizarQuantidades(in []float64) ([]float64, bool) {
	if len(in) == 0 {
		return nil, false
	}
	for _, q := range in {
		if q < 0 {
			return nil, false
		}
	}
	sorted := append([]float64(nil), in...)
	sort.Float64s(sorted)
	out := make([]float64, 0, len(sorted))
	var prev float64
	for i, q := range sorted {
		if i > 0 && q == prev {
			continue
		}
		out = append(out, q)
		prev = q
	}
	return out, true
}

func validarPromocao(c CandidatoOferta) *FalhaExtracao {
	p := c.Promocao
	if p == nil {
		return nil
	}
	if p.ValorPromocional <= 0 {
		return &FalhaExtracao{Codigo: CodigoPromocaoInvalida, Detalhe: "valorPromocional deve ser > 0", Candidato: c}
	}

	// Canal: at most one of cartão/clube (ADR 0032).
	cartao := p.PromocaoCartao != nil
	clube := p.PromocaoClube != nil
	if cartao && clube {
		return &FalhaExtracao{Codigo: CodigoPromocaoInvalida, Detalhe: "promocaoCartao e promocaoClube são mutuamente exclusivos", Candidato: c}
	}
	if cartao && !*p.PromocaoCartao {
		return &FalhaExtracao{Codigo: CodigoPromocaoInvalida, Detalhe: "promocaoCartao deve ser true", Candidato: c}
	}
	if clube && !*p.PromocaoClube {
		return &FalhaExtracao{Codigo: CodigoPromocaoInvalida, Detalhe: "promocaoClube deve ser true", Candidato: c}
	}
	temCanal := cartao || clube

	// Mecânica: at most one of leve/pague or quantidadePromocao.
	levePague := p.Leve != nil || p.Pague != nil
	qtd := p.QuantidadePromocao != nil
	if levePague && qtd {
		return &FalhaExtracao{Codigo: CodigoPromocaoInvalida, Detalhe: "leve/pague e quantidadePromocao são mutuamente exclusivos", Candidato: c}
	}
	if levePague {
		if p.Leve == nil || p.Pague == nil || *p.Leve <= 0 || *p.Pague <= 0 {
			return &FalhaExtracao{Codigo: CodigoPromocaoInvalida, Detalhe: "leve/pague inválidos", Candidato: c}
		}
	}
	if qtd && *p.QuantidadePromocao <= 0 {
		return &FalhaExtracao{Codigo: CodigoPromocaoInvalida, Detalhe: "quantidadePromocao inválida", Candidato: c}
	}
	temMecanica := levePague || qtd

	if !temCanal && !temMecanica {
		return &FalhaExtracao{Codigo: CodigoPromocaoInvalida, Detalhe: "promocao exige canal e/ou mecânica de quantidade", Candidato: c}
	}
	return nil
}

func validarComparativo(c CandidatoOferta, quantidades []float64) *FalhaExtracao {
	comp := c.Comparativo
	if comp == nil {
		return nil
	}
	if comp.Quantidade <= 0 {
		return &FalhaExtracao{Codigo: CodigoComparativoInvalido, Detalhe: "comparativo.quantidade deve ser > 0", Candidato: c}
	}
	if comp.Valor <= 0 {
		return &FalhaExtracao{Codigo: CodigoComparativoInvalido, Detalhe: "comparativo.valor deve ser > 0", Candidato: c}
	}
	// quantidades is normalized ascending; badge must be a strict fraction of the pack.
	if comp.Quantidade >= quantidades[0] {
		return &FalhaExtracao{Codigo: CodigoComparativoInvalido, Detalhe: "comparativo.quantidade deve ser < menor quantidade do pack", Candidato: c}
	}
	return nil
}

func normalizarCategorias(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
		n := NormalizarRotulo(raw)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}
