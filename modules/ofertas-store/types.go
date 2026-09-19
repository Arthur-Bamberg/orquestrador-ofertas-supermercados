package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

type MercadoID string
type FonteID string
type ProdutoID string
type MarcaID string
type DocumentoID string
type ColetaID string
type OfertaID string

type Mercado struct {
	ID   MercadoID `json:"id"`
	Nome string    `json:"nome"`
}

type TipoFonte string

const (
	TipoEncarte TipoFonte = "encarte"
	TipoSite    TipoFonte = "site"
)

type Fonte struct {
	ID                  FonteID   `json:"id"`
	MercadoID           MercadoID `json:"mercadoId"`
	URL                 string    `json:"url"`
	FiltroNomeDocumento string    `json:"filtroNomeDocumento"`
	// Ativa: nil means active (default). Job diário and discover skip when false.
	Ativa *bool `json:"ativa,omitempty"`
	// Tipo: empty means encarte (PDF pipeline). site Fontes stay in the catalog for Coleta.
	Tipo TipoFonte `json:"tipo,omitempty"`
}

// Bool returns a *bool for Fonte.Ativa and similar optional flags.
func Bool(v bool) *bool { return &v }

// IsAtiva reports whether the Fonte participates in discovery/job (default true).
func (f Fonte) IsAtiva() bool {
	return f.Ativa == nil || *f.Ativa
}

// TipoOuEncarte is the persisted collection kind (default encarte).
func (f Fonte) TipoOuEncarte() TipoFonte {
	if f.Tipo == "" {
		return TipoEncarte
	}
	return f.Tipo
}

type Produto struct {
	ID           ProdutoID `json:"id"`
	Nome         string    `json:"nome"`
	OriginalNome string    `json:"originalNome"`
	NomeNorm     string    `json:"nomeNorm"`
	Categorias   []string  `json:"categorias"`
}

type Marca struct {
	ID       MarcaID `json:"id"`
	Nome     string  `json:"nome"`
	NomeNorm string  `json:"nomeNorm"`
}

type EstadoDocumento string

const (
	EstadoProcessando EstadoDocumento = "processando"
	EstadoConcluido   EstadoDocumento = "concluido"
	EstadoParcial     EstadoDocumento = "parcial"
	EstadoFalhou      EstadoDocumento = "falhou"
)

type Documento struct {
	ID                DocumentoID     `json:"id"`
	FonteID           FonteID         `json:"fonteId"`
	MercadoID         MercadoID       `json:"mercadoId"`
	Filename          string          `json:"filename"`
	Dia               string          `json:"dia"`
	Estado            EstadoDocumento `json:"estado"`
	Fingerprint       string          `json:"fingerprint,omitempty"`
	ConteudoIdenticoA *DocumentoID    `json:"conteudoIdenticoA,omitempty"`
	UltimoErro        string          `json:"ultimoErro,omitempty"`
	Atualizado        time.Time       `json:"atualizado"`
}

type Coleta struct {
	ID         ColetaID        `json:"id"`
	Termo      string          `json:"termo"`
	MercadoID  MercadoID       `json:"mercadoId"`
	Dia        string          `json:"dia"`
	Estado     EstadoDocumento `json:"estado"`
	UltimoErro string          `json:"ultimoErro,omitempty"`
	Atualizado time.Time       `json:"atualizado"`
}

type OrigemData string

const (
	OrigemExtrator           OrigemData = "extrator"
	OrigemFilename           OrigemData = "filename"
	OrigemPrimeiraDescoberta OrigemData = "primeiraDescoberta"
	OrigemColeta             OrigemData = "coleta"
)

type Medida string

const (
	MedidaG       Medida = "g"
	MedidaML      Medida = "ml"
	MedidaUnidade Medida = "unidade"
)

type Promocao struct {
	Leve               *float64 `json:"leve,omitempty"`
	Pague              *float64 `json:"pague,omitempty"`
	QuantidadePromocao *float64 `json:"quantidadePromocao,omitempty"`
	PromocaoCartao     *bool    `json:"promocaoCartao,omitempty"`
	PromocaoClube      *bool    `json:"promocaoClube,omitempty"`
	ValorPromocional   float64  `json:"valorPromocional"`
}

type Comparativo struct {
	Quantidade float64 `json:"quantidade"`
	Valor      float64 `json:"valor"`
}

type Oferta struct {
	ID                   OfertaID     `json:"id"`
	DocumentoID          DocumentoID  `json:"documentoId,omitempty"`
	ProdutoID            ProdutoID    `json:"produtoId"`
	MarcaID              *MarcaID     `json:"marcaId,omitempty"`
	MercadoID            MercadoID    `json:"mercadoId"`
	Valor                float64      `json:"valor"`
	Quantidades          []float64    `json:"quantidades"`
	Medida               Medida       `json:"medida"`
	DataInicio           string       `json:"dataInicio"`
	DataExpiracao        string       `json:"dataExpiracao"`
	OrigemDataInicio     OrigemData   `json:"origemDataInicio"`
	OrigemDataExpiracao  OrigemData   `json:"origemDataExpiracao"`
	Promocao             *Promocao    `json:"promocao,omitempty"`
	Comparativo          *Comparativo `json:"comparativo,omitempty"`
	IndicacaoPromocional bool         `json:"indicacaoPromocional"`
}

// UnmarshalJSON accepts quantidades[] and legacy singular quantidade.
func (o *Oferta) UnmarshalJSON(data []byte) error {
	var j struct {
		ID                   OfertaID     `json:"id"`
		DocumentoID          DocumentoID  `json:"documentoId"`
		ProdutoID            ProdutoID    `json:"produtoId"`
		MarcaID              *MarcaID     `json:"marcaId,omitempty"`
		MercadoID            MercadoID    `json:"mercadoId"`
		Valor                float64      `json:"valor"`
		Quantidades          []float64    `json:"quantidades"`
		Quantidade           *float64     `json:"quantidade"`
		Medida               Medida       `json:"medida"`
		DataInicio           string       `json:"dataInicio"`
		DataExpiracao        string       `json:"dataExpiracao"`
		OrigemDataInicio     OrigemData   `json:"origemDataInicio"`
		OrigemDataExpiracao  OrigemData   `json:"origemDataExpiracao"`
		Promocao             *Promocao    `json:"promocao,omitempty"`
		Comparativo          *Comparativo `json:"comparativo,omitempty"`
		IndicacaoPromocional bool         `json:"indicacaoPromocional"`
	}
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	o.ID = j.ID
	o.DocumentoID = j.DocumentoID
	o.ProdutoID = j.ProdutoID
	o.MarcaID = j.MarcaID
	o.MercadoID = j.MercadoID
	o.Valor = j.Valor
	o.Quantidades = coalesceQuantidades(j.Quantidades, j.Quantidade)
	o.Medida = j.Medida
	o.DataInicio = j.DataInicio
	o.DataExpiracao = j.DataExpiracao
	o.OrigemDataInicio = j.OrigemDataInicio
	o.OrigemDataExpiracao = j.OrigemDataExpiracao
	o.Promocao = j.Promocao
	o.Comparativo = j.Comparativo
	o.IndicacaoPromocional = j.IndicacaoPromocional
	return nil
}

type CandidatoOferta struct {
	Produto       string       `json:"produto"`
	Marca         string       `json:"marca,omitempty"`
	Categorias    []string     `json:"categorias,omitempty"`
	Valor         float64      `json:"valor"`
	Quantidades   []float64    `json:"quantidades"`
	Medida        string       `json:"medida"`
	DataInicio    string       `json:"dataInicio"`
	DataExpiracao string       `json:"dataExpiracao"`
	Promocao      *Promocao    `json:"promocao,omitempty"`
	Comparativo   *Comparativo `json:"comparativo,omitempty"`
}

// UnmarshalJSON accepts quantidades[] and legacy singular quantidade.
func (c *CandidatoOferta) UnmarshalJSON(data []byte) error {
	var j struct {
		Produto       string       `json:"produto"`
		Marca         string       `json:"marca,omitempty"`
		Categorias    []string     `json:"categorias,omitempty"`
		Valor         float64      `json:"valor"`
		Quantidades   []float64    `json:"quantidades"`
		Quantidade    *float64     `json:"quantidade"`
		Medida        string       `json:"medida"`
		DataInicio    string       `json:"dataInicio"`
		DataExpiracao string       `json:"dataExpiracao"`
		Promocao      *Promocao    `json:"promocao,omitempty"`
		Comparativo   *Comparativo `json:"comparativo,omitempty"`
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
	c.Comparativo = j.Comparativo
	return nil
}

type FalhaExtracao struct {
	Codigo    string          `json:"codigo"`
	Detalhe   string          `json:"detalhe,omitempty"`
	Candidato CandidatoOferta `json:"candidato"`
}

type UsoExtratorPagina struct {
	Page         int   `json:"page"`
	PromptTokens int64 `json:"promptTokens"`
	CacheTokens  int64 `json:"cacheTokens"`
	OutputTokens int64 `json:"outputTokens"`
}

type UsoExtrator struct {
	DocumentoID  DocumentoID         `json:"documentoId,omitempty"`
	Tentativa    string              `json:"tentativa,omitempty"`
	ArtefatoPath string              `json:"artefatoPath"`
	Provider     string              `json:"provider,omitempty"`
	Model        string              `json:"model"`
	PromptTokens int64               `json:"promptTokens"`
	CacheTokens  int64               `json:"cacheTokens"`
	OutputTokens int64               `json:"outputTokens"`
	Paginas      []UsoExtratorPagina `json:"paginas,omitempty"`
}

type PageImage struct {
	Page int
	JPEG []byte
}

type PDFDescoberto struct {
	Filename string
	URL      string
}

type FilenameVigencia struct {
	DataInicio    string
	DataExpiracao string
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

func FingerprintPDF(pdf []byte) string {
	sum := sha256.Sum256(pdf)
	return hex.EncodeToString(sum[:])
}

func EstadoAposValidacao(ofertasValidas, falhas int) EstadoDocumento {
	if ofertasValidas == 0 {
		return EstadoFalhou
	}
	if falhas > 0 {
		return EstadoParcial
	}
	return EstadoConcluido
}

func DeveReprocessar(estado EstadoDocumento) bool {
	switch estado {
	case EstadoConcluido, EstadoParcial:
		return false
	default:
		return true
	}
}

// NormalizeQuantidadesForKey keeps admin-created Ofertas stable when callers
// send the same discrete quantities in a different order.
func NormalizeQuantidadesForKey(in []float64) []float64 {
	if len(in) == 0 {
		return nil
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
	return out
}
