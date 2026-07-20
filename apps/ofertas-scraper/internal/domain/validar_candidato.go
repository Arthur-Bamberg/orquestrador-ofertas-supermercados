package domain

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type Medida string

const (
	MedidaG       Medida = "g"
	MedidaML      Medida = "ml"
	MedidaUnidade Medida = "unidade"
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
)

// CandidatoOferta is the Extrator candidate before match-or-create.
type CandidatoOferta struct {
	Produto       string    `json:"produto"`
	Marca         string    `json:"marca,omitempty"`
	Categorias    []string  `json:"categorias,omitempty"`
	Valor         float64   `json:"valor"`
	Quantidades   []float64 `json:"quantidades"`
	Medida        string    `json:"medida"`
	DataInicio    string    `json:"dataInicio"`
	DataExpiracao string    `json:"dataExpiracao"`
	Promocao      *Promocao `json:"promocao,omitempty"`
}

// UnmarshalJSON accepts quantidades[] and legacy singular quantidade (ADR 0030).
func (c *CandidatoOferta) UnmarshalJSON(data []byte) error {
	var j struct {
		Produto       string    `json:"produto"`
		Marca         string    `json:"marca,omitempty"`
		Categorias    []string  `json:"categorias,omitempty"`
		Valor         float64   `json:"valor"`
		Quantidades   []float64 `json:"quantidades"`
		Quantidade    *float64  `json:"quantidade"`
		Medida        string    `json:"medida"`
		DataInicio    string    `json:"dataInicio"`
		DataExpiracao string    `json:"dataExpiracao"`
		Promocao      *Promocao `json:"promocao,omitempty"`
	}
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	c.Produto = j.Produto
	c.Marca = j.Marca
	c.Categorias = j.Categorias
	c.Valor = j.Valor
	c.Quantidades = coalesceQuantidades(j.Quantidades, j.Quantidade)
	c.Medida = j.Medida
	c.DataInicio = j.DataInicio
	c.DataExpiracao = j.DataExpiracao
	c.Promocao = j.Promocao
	return nil
}

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
}

type Promocao struct {
	Leve               *float64 `json:"leve,omitempty"`
	Pague              *float64 `json:"pague,omitempty"`
	QuantidadePromocao *float64 `json:"quantidadePromocao,omitempty"`
	PromocaoCartao     *bool    `json:"promocaoCartao,omitempty"`
	PromocaoClube      *bool    `json:"promocaoClube,omitempty"`
	ValorPromocional   float64  `json:"valorPromocional"`
}

type FalhaExtracao struct {
	Codigo    string          `json:"codigo"`
	Detalhe   string          `json:"detalhe,omitempty"`
	Candidato CandidatoOferta `json:"candidato"`
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

	levePague := p.Leve != nil || p.Pague != nil
	qtd := p.QuantidadePromocao != nil
	cartao := p.PromocaoCartao != nil
	clube := p.PromocaoClube != nil
	n := 0
	if levePague {
		n++
	}
	if qtd {
		n++
	}
	if cartao {
		n++
	}
	if clube {
		n++
	}
	if n != 1 {
		return &FalhaExtracao{Codigo: CodigoPromocaoInvalida, Detalhe: "promocao deve ser exatamente um dos quatro formatos", Candidato: c}
	}

	switch {
	case levePague:
		if p.Leve == nil || p.Pague == nil || *p.Leve <= 0 || *p.Pague <= 0 {
			return &FalhaExtracao{Codigo: CodigoPromocaoInvalida, Detalhe: "leve/pague inválidos", Candidato: c}
		}
	case qtd:
		if *p.QuantidadePromocao <= 0 {
			return &FalhaExtracao{Codigo: CodigoPromocaoInvalida, Detalhe: "quantidadePromocao inválida", Candidato: c}
		}
	case cartao:
		if !*p.PromocaoCartao {
			return &FalhaExtracao{Codigo: CodigoPromocaoInvalida, Detalhe: "promocaoCartao deve ser true", Candidato: c}
		}
	case clube:
		if !*p.PromocaoClube {
			return &FalhaExtracao{Codigo: CodigoPromocaoInvalida, Detalhe: "promocaoClube deve ser true", Candidato: c}
		}
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
