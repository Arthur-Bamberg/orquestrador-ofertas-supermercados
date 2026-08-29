package domain_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

func TestValidarCandidato_aceitaCandidatoValido(t *testing.T) {
	c := domain.CandidatoOferta{
		Produto:     "Arroz integral",
		Valor:       12.9,
		Quantidades: []float64{1000},
		Medida:      "g",
		DataInicio:  "2026-07-18", DataExpiracao: "2026-07-20",
		Categorias: []string{"mercearia", "arroz"},
	}

	oferta, falha := domain.ValidarCandidato(c)
	if falha != nil {
		t.Fatalf("esperava sem Falha, obteve %#v", falha)
	}
	if oferta.Produto != "Arroz integral" {
		t.Fatalf("produto: got %q", oferta.Produto)
	}
	if oferta.Medida != domain.MedidaG {
		t.Fatalf("medida: got %q", oferta.Medida)
	}
}

func TestValidarCandidato_rejeitaMedidaNaoNormalizada(t *testing.T) {
	c := domain.CandidatoOferta{
		Produto:     "Arroz integral",
		Valor:       12.9,
		Quantidades: []float64{1},
		Medida:      "kg",
		DataInicio:  "2026-07-18", DataExpiracao: "2026-07-20",
	}

	_, falha := domain.ValidarCandidato(c)
	if falha == nil {
		t.Fatal("esperava Falha de Extração para medida kg")
	}
	if falha.Codigo != domain.CodigoMedidaInvalida {
		t.Fatalf("codigo: got %q want %q", falha.Codigo, domain.CodigoMedidaInvalida)
	}
	if falha.Candidato.Medida != "kg" {
		t.Fatalf("candidato deve preservar medida rejeitada, got %q", falha.Candidato.Medida)
	}
}

func TestValidarCandidato_rejeitaProdutoVazio(t *testing.T) {
	c := domain.CandidatoOferta{
		Produto:     "  ",
		Valor:       1,
		Quantidades: []float64{1},
		Medida:      "unidade",
		DataInicio:  "2026-07-18", DataExpiracao: "2026-07-20",
	}

	_, falha := domain.ValidarCandidato(c)
	if falha == nil {
		t.Fatal("esperava Falha para produto vazio")
	}
	if falha.Codigo != domain.CodigoProdutoInvalido {
		t.Fatalf("codigo: got %q", falha.Codigo)
	}
}

func TestValidarCandidato_rejeitaValorNaoPositivo(t *testing.T) {
	c := domain.CandidatoOferta{
		Produto:     "Banana",
		Valor:       0,
		Quantidades: []float64{1},
		Medida:      "unidade",
		DataInicio:  "2026-07-18", DataExpiracao: "2026-07-20",
	}

	_, falha := domain.ValidarCandidato(c)
	if falha == nil || falha.Codigo != domain.CodigoValorInvalido {
		t.Fatalf("esperava valor_invalido, got %#v", falha)
	}
}

func TestValidarCandidato_aceitaDataExpiracaoPassada(t *testing.T) {
	c := domain.CandidatoOferta{
		Produto:     "Banana",
		Valor:       2.5,
		Quantidades: []float64{1},
		Medida:      "unidade",
		DataInicio:  "2020-01-01", DataExpiracao: "2020-01-01",
	}

	oferta, falha := domain.ValidarCandidato(c)
	if falha != nil {
		t.Fatalf("data passada deve ser aceita para histórico: %#v", falha)
	}
	if oferta.DataExpiracao != "2020-01-01" {
		t.Fatalf("data: got %q", oferta.DataExpiracao)
	}
}

func TestValidarCandidato_rejeitaDataExpiracaoInvalida(t *testing.T) {
	c := domain.CandidatoOferta{
		Produto:     "Banana",
		Valor:       2.5,
		Quantidades: []float64{1},
		Medida:      "unidade",
		DataInicio:  "2020-01-01", DataExpiracao: "20/01/2020",
	}

	_, falha := domain.ValidarCandidato(c)
	if falha == nil || falha.Codigo != domain.CodigoDataExpiracaoInvalida {
		t.Fatalf("esperava data_expiracao_invalida, got %#v", falha)
	}
}

func TestValidarCandidato_rejeitaInicioAposExpiracao(t *testing.T) {
	c := domain.CandidatoOferta{
		Produto:       "Banana",
		Valor:         2.5,
		Quantidades:   []float64{1},
		Medida:        "unidade",
		DataInicio:    "2026-07-25",
		DataExpiracao: "2026-07-20",
	}

	_, falha := domain.ValidarCandidato(c)
	if falha == nil || falha.Codigo != domain.CodigoVigenciaInvalida {
		t.Fatalf("esperava vigencia_invalida, got %#v", falha)
	}
}

func TestValidarCandidato_aceitaVigenciaUmDia(t *testing.T) {
	c := domain.CandidatoOferta{
		Produto:       "Banana",
		Valor:         2.5,
		Quantidades:   []float64{1},
		Medida:        "unidade",
		DataInicio:    "2026-07-20",
		DataExpiracao: "2026-07-20",
	}
	if _, falha := domain.ValidarCandidato(c); falha != nil {
		t.Fatalf("mesmo dia deve ser válido: %#v", falha)
	}
}

func TestValidarCandidato_rejeitaDataInicioInvalida(t *testing.T) {
	c := domain.CandidatoOferta{
		Produto:       "Banana",
		Valor:         2.5,
		Quantidades:   []float64{1},
		Medida:        "unidade",
		DataInicio:    "20/07/2026",
		DataExpiracao: "2026-07-20",
	}
	_, falha := domain.ValidarCandidato(c)
	if falha == nil || falha.Codigo != domain.CodigoDataInicioInvalida {
		t.Fatalf("esperava data_inicio_invalida, got %#v", falha)
	}
}

func TestValidarCandidato_rejeitaDataExpiracaoVazia(t *testing.T) {
	c := domain.CandidatoOferta{
		Produto:     "Banana",
		Valor:       2.5,
		Quantidades: []float64{1},
		Medida:      "unidade",
		DataInicio:  "2026-07-18",
	}
	_, falha := domain.ValidarCandidato(c)
	if falha == nil || falha.Codigo != domain.CodigoDataExpiracaoInvalida {
		t.Fatalf("esperava data_expiracao_invalida, got %#v", falha)
	}
}

func TestValidarCandidato_aceitaPromocaoClube(t *testing.T) {
	clube := true
	c := domain.CandidatoOferta{
		Produto:       "Leite",
		Valor:         5,
		Quantidades:   []float64{1000},
		Medida:        "ml",
		DataInicio:    "2026-07-18",
		DataExpiracao: "2026-07-20",
		Promocao: &domain.Promocao{
			PromocaoClube:    &clube,
			ValorPromocional: 4,
		},
	}

	oferta, falha := domain.ValidarCandidato(c)
	if falha != nil {
		t.Fatalf("promocao clube válida: %#v", falha)
	}
	if oferta.Promocao == nil || oferta.Promocao.PromocaoClube == nil || !*oferta.Promocao.PromocaoClube {
		t.Fatalf("promocao: %#v", oferta.Promocao)
	}
}

func TestValidarCandidato_rejeitaPromocaoCartaoEClube(t *testing.T) {
	cartao, clube := true, true
	c := domain.CandidatoOferta{
		Produto:       "Leite",
		Valor:         5,
		Quantidades:   []float64{1000},
		Medida:        "ml",
		DataInicio:    "2026-07-18",
		DataExpiracao: "2026-07-20",
		Promocao: &domain.Promocao{
			PromocaoCartao:   &cartao,
			PromocaoClube:    &clube,
			ValorPromocional: 4,
		},
	}
	_, falha := domain.ValidarCandidato(c)
	if falha == nil || falha.Codigo != domain.CodigoPromocaoInvalida {
		t.Fatalf("esperava promocao_invalida, got %#v", falha)
	}
}

func TestValidarCandidato_aceitaPromocaoCanalEMecanicaCompostos(t *testing.T) {
	clube := true
	leve, pague := 12.0, 10.0
	c := domain.CandidatoOferta{
		Produto:       "Cerveja Lata",
		Marca:         "Brahma",
		Valor:         4.99,
		Quantidades:   []float64{473},
		Medida:        "ml",
		DataInicio:    "2026-07-18",
		DataExpiracao: "2026-07-20",
		Promocao: &domain.Promocao{
			Leve:             &leve,
			Pague:            &pague,
			PromocaoClube:    &clube,
			ValorPromocional: 4.16,
		},
	}
	oferta, falha := domain.ValidarCandidato(c)
	if falha != nil {
		t.Fatalf("composicao canal+mecanica válida: %#v", falha)
	}
	if oferta.Promocao == nil || oferta.Promocao.Leve == nil || oferta.Promocao.PromocaoClube == nil {
		t.Fatalf("promocao: %#v", oferta.Promocao)
	}
}

func TestValidarCandidato_marcaOpcional(t *testing.T) {
	c := domain.CandidatoOferta{
		Produto:     "Banana",
		Valor:       2.5,
		Quantidades: []float64{1},
		Medida:      "unidade",
		DataInicio:  "2026-07-18", DataExpiracao: "2026-07-20",
	}

	oferta, falha := domain.ValidarCandidato(c)
	if falha != nil {
		t.Fatalf("marca omitida deve ser válida: %#v", falha)
	}
	if oferta.Marca != "" {
		t.Fatalf("marca: got %q", oferta.Marca)
	}
}

func TestValidarCandidato_aceitaComparativo(t *testing.T) {
	c := domain.CandidatoOferta{
		Produto:       "Sabão em pó",
		Marca:         "Girando Sol",
		Valor:         24.90,
		Quantidades:   []float64{4000},
		Medida:        "g",
		DataInicio:    "2026-07-18",
		DataExpiracao: "2026-07-20",
		Comparativo: &domain.Comparativo{
			Quantidade: 800,
			Valor:      4.98,
		},
	}
	oferta, falha := domain.ValidarCandidato(c)
	if falha != nil {
		t.Fatalf("comparativo válido: %#v", falha)
	}
	if oferta.Comparativo == nil || oferta.Comparativo.Quantidade != 800 || oferta.Comparativo.Valor != 4.98 {
		t.Fatalf("comparativo: %#v", oferta.Comparativo)
	}
}

func TestValidarCandidato_rejeitaComparativoQuantidadeNaoMenorQuePack(t *testing.T) {
	c := domain.CandidatoOferta{
		Produto: "Sabão em pó", Valor: 24.90, Quantidades: []float64{4000}, Medida: "g",
		DataInicio: "2026-07-18", DataExpiracao: "2026-07-20",
		Comparativo: &domain.Comparativo{Quantidade: 4000, Valor: 24.90},
	}
	_, falha := domain.ValidarCandidato(c)
	if falha == nil || falha.Codigo != domain.CodigoComparativoInvalido {
		t.Fatalf("esperava comparativo_invalido, got %#v", falha)
	}
}

func TestValidarCandidato_rejeitaComparativoValorNaoPositivo(t *testing.T) {
	c := domain.CandidatoOferta{
		Produto: "Sabão em pó", Valor: 24.90, Quantidades: []float64{4000}, Medida: "g",
		DataInicio: "2026-07-18", DataExpiracao: "2026-07-20",
		Comparativo: &domain.Comparativo{Quantidade: 800, Valor: 0},
	}
	_, falha := domain.ValidarCandidato(c)
	if falha == nil || falha.Codigo != domain.CodigoComparativoInvalido {
		t.Fatalf("esperava comparativo_invalido, got %#v", falha)
	}
}

func TestValidarCandidato_aceitaComparativoComPromocao(t *testing.T) {
	clube := true
	c := domain.CandidatoOferta{
		Produto: "Sabão em pó", Valor: 24.90, Quantidades: []float64{4000}, Medida: "g",
		DataInicio: "2026-07-18", DataExpiracao: "2026-07-20",
		Promocao:    &domain.Promocao{PromocaoClube: &clube, ValorPromocional: 22.90},
		Comparativo: &domain.Comparativo{Quantidade: 800, Valor: 4.98},
	}
	oferta, falha := domain.ValidarCandidato(c)
	if falha != nil {
		t.Fatalf("comparativo+promocao: %#v", falha)
	}
	if oferta.Promocao == nil || oferta.Comparativo == nil {
		t.Fatalf("ambos devem persistir: promo=%#v comp=%#v", oferta.Promocao, oferta.Comparativo)
	}
}

func TestValidarCandidato_aceitaPromocaoLevePague(t *testing.T) {
	leve, pague := 3.0, 2.0
	c := domain.CandidatoOferta{
		Produto:     "Refrigerante",
		Marca:       "Cola",
		Valor:       8,
		Quantidades: []float64{2000},
		Medida:      "ml",
		DataInicio:  "2026-07-18", DataExpiracao: "2026-07-20",
		Promocao: &domain.Promocao{
			Leve:             &leve,
			Pague:            &pague,
			ValorPromocional: 12,
		},
	}

	oferta, falha := domain.ValidarCandidato(c)
	if falha != nil {
		t.Fatalf("promocao válida: %#v", falha)
	}
	if oferta.Promocao == nil || oferta.Promocao.ValorPromocional != 12 {
		t.Fatalf("promocao: %#v", oferta.Promocao)
	}
}

func TestValidarCandidato_normalizaQuantidadesDedupSort(t *testing.T) {
	c := domain.CandidatoOferta{
		Produto: "Barra de chocolate", Valor: 5.99,
		Quantidades: []float64{200, 90, 90, 150}, Medida: "g",
		DataInicio: "2026-07-18", DataExpiracao: "2026-07-20",
	}
	oferta, falha := domain.ValidarCandidato(c)
	if falha != nil {
		t.Fatalf("esperava ok: %#v", falha)
	}
	want := []float64{90, 150, 200}
	if !reflect.DeepEqual(oferta.Quantidades, want) {
		t.Fatalf("quantidades: got %#v want %#v", oferta.Quantidades, want)
	}
}

func TestValidarCandidato_rejeitaQuantidadesVazias(t *testing.T) {
	c := domain.CandidatoOferta{
		Produto: "X", Valor: 1, Quantidades: nil, Medida: "unidade",
		DataInicio: "2026-07-18", DataExpiracao: "2026-07-20",
	}
	_, falha := domain.ValidarCandidato(c)
	if falha == nil || falha.Codigo != domain.CodigoQuantidadeInvalida {
		t.Fatalf("esperava quantidade_invalida, got %#v", falha)
	}
}

func TestCandidatoOferta_UnmarshalJSON_legadoQuantidade(t *testing.T) {
	raw := []byte(`{"produto":"Arroz","valor":10,"quantidade":1000,"medida":"g","dataInicio":"2026-07-18","dataExpiracao":"2026-07-20"}`)
	var c domain.CandidatoOferta
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c.Quantidades, []float64{1000}) {
		t.Fatalf("got %#v", c.Quantidades)
	}
}

func TestOferta_UnmarshalJSON_legadoQuantidade(t *testing.T) {
	raw := []byte(`{"id":"o1","documentoId":"d1","produtoId":"p1","mercadoId":"m1","valor":5,"quantidade":90,"medida":"g","dataInicio":"2026-07-18","dataExpiracao":"2026-07-20","origemDataInicio":"extrator","origemDataExpiracao":"extrator"}`)
	var o domain.Oferta
	if err := json.Unmarshal(raw, &o); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(o.Quantidades, []float64{90}) {
		t.Fatalf("got %#v", o.Quantidades)
	}
}
