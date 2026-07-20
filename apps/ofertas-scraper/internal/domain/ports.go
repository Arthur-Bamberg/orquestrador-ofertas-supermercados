package domain

import (
	"context"
	"time"
)

type MercadoID string
type FonteID string
type ProdutoID string
type MarcaID string
type DocumentoID string
type OfertaID string

type Mercado struct {
	ID   MercadoID `json:"id"`
	Nome string    `json:"nome"`
}

type Fonte struct {
	ID                  FonteID   `json:"id"`
	MercadoID           MercadoID `json:"mercadoId"`
	URL                 string    `json:"url"`
	FiltroNomeDocumento string    `json:"filtroNomeDocumento"`
}

type Produto struct {
	ID         ProdutoID `json:"id"`
	Nome       string    `json:"nome"`
	NomeNorm   string    `json:"nomeNorm"`
	Categorias []string  `json:"categorias"`
}

type Marca struct {
	ID       MarcaID `json:"id"`
	Nome     string  `json:"nome"`
	NomeNorm string  `json:"nomeNorm"`
}

type Documento struct {
	ID         DocumentoID     `json:"id"`
	FonteID    FonteID         `json:"fonteId"`
	MercadoID  MercadoID       `json:"mercadoId"`
	Filename   string          `json:"filename"`
	Dia        string          `json:"dia"`
	Estado     EstadoDocumento `json:"estado"`
	UltimoErro string          `json:"ultimoErro,omitempty"`
	Atualizado time.Time       `json:"atualizado"`
}

// OrigemData records how a vigência date was obtained (ADR 0028).
type OrigemData string

const (
	OrigemExtrator           OrigemData = "extrator"
	OrigemFilename           OrigemData = "filename"
	OrigemPrimeiraDescoberta OrigemData = "primeiraDescoberta"
)

// Oferta is the persisted price observation (after match-or-create).
type Oferta struct {
	ID                  OfertaID     `json:"id"`
	DocumentoID         DocumentoID  `json:"documentoId"`
	ProdutoID           ProdutoID    `json:"produtoId"`
	MarcaID             *MarcaID     `json:"marcaId,omitempty"`
	MercadoID           MercadoID    `json:"mercadoId"`
	Valor               float64      `json:"valor"`
	Quantidades         []float64    `json:"quantidades"`
	Medida              Medida       `json:"medida"`
	DataInicio          string       `json:"dataInicio"`
	DataExpiracao       string       `json:"dataExpiracao"`
	OrigemDataInicio    OrigemData   `json:"origemDataInicio"`
	OrigemDataExpiracao OrigemData   `json:"origemDataExpiracao"`
	Promocao            *Promocao    `json:"promocao,omitempty"`
	Comparativo         *Comparativo `json:"comparativo,omitempty"`
}

type MercadoRepository interface {
	List(ctx context.Context) ([]Mercado, error)
	Save(ctx context.Context, m Mercado) error
	Get(ctx context.Context, id MercadoID) (Mercado, error)
}

type FonteRepository interface {
	List(ctx context.Context) ([]Fonte, error)
	Save(ctx context.Context, f Fonte) error
}

type ProdutoRepository interface {
	GetByNomeNorm(ctx context.Context, nomeNorm string) (Produto, bool, error)
	Save(ctx context.Context, p Produto) error
}

type MarcaRepository interface {
	GetByNomeNorm(ctx context.Context, nomeNorm string) (Marca, bool, error)
	Save(ctx context.Context, m Marca) error
}

type DocumentoRepository interface {
	GetByIdentity(ctx context.Context, fonteID FonteID, filename, dia string) (Documento, bool, error)
	Save(ctx context.Context, d Documento) error
	// EarliestDia is the smallest discovery day for Fonte+filename (any Documento state).
	EarliestDia(ctx context.Context, fonteID FonteID, filename string) (dia string, ok bool, err error)
}

type OfertaRepository interface {
	SaveAll(ctx context.Context, documentoID DocumentoID, ofertas []Oferta) error
	ListByDocumento(ctx context.Context, documentoID DocumentoID) ([]Oferta, error)
	// ListDocumentoIDsByProduto returns Documento ids indexed under ofertas:produto:{produtoId} (ADR 0026).
	ListDocumentoIDsByProduto(ctx context.Context, produtoID ProdutoID) ([]DocumentoID, error)
}

type FalhaExtracaoRepository interface {
	SaveAll(ctx context.Context, documentoID DocumentoID, falhas []FalhaExtracao) error
}

type PageImage struct {
	Page int
	JPEG []byte
}

// UsoExtratorPagina is per-page token usage inside one Extract call (ADR 0031/0033).
type UsoExtratorPagina struct {
	Page         int   `json:"page"`
	PromptTokens int64 `json:"promptTokens"`
	CacheTokens  int64 `json:"cacheTokens"`
	OutputTokens int64 `json:"outputTokens"`
}

// UsoExtrator records Extrator token consumption for one processing tentativa (ADR 0033).
type UsoExtrator struct {
	DocumentoID  DocumentoID         `json:"documentoId,omitempty"`
	Tentativa    string              `json:"tentativa,omitempty"`
	ArtefatoPath string              `json:"artefatoPath"`
	Model        string              `json:"model"`
	PromptTokens int64               `json:"promptTokens"`
	CacheTokens  int64               `json:"cacheTokens"`
	OutputTokens int64               `json:"outputTokens"`
	Paginas      []UsoExtratorPagina `json:"paginas,omitempty"`
}

type Extrator interface {
	// Extract returns candidatos, raw JSON, and optional Uso (nil when the adapter has no usage).
	Extract(ctx context.Context, images []PageImage) (candidatos []CandidatoOferta, raw []byte, uso *UsoExtrator, err error)
}

type UsoExtratorRepository interface {
	Save(ctx context.Context, uso UsoExtrator) error
}

// PDFDescoberto is a PDF link found on a Fonte page (ADR 0020).
type PDFDescoberto struct {
	Filename string // last path segment — Documento identity
	URL      string // absolute download URL
}

type ArtefatoStore interface {
	// AttemptPath is the directory for a Documento tentativa (relative or absolute under the store root).
	AttemptPath(doc Documento, tentativa string) string
	SavePDF(ctx context.Context, doc Documento, tentativa string, pdf []byte) error
	SaveImages(ctx context.Context, doc Documento, tentativa string, images []PageImage) error
	SaveRawExtrator(ctx context.Context, doc Documento, tentativa string, raw []byte) error
	SaveUsoExtrator(ctx context.Context, doc Documento, tentativa string, uso UsoExtrator) error
	SaveValidated(ctx context.Context, doc Documento, tentativa string, ofertas []Oferta, falhas []FalhaExtracao) error
}

type FonteClient interface {
	DiscoverPDFs(ctx context.Context, fonte Fonte) ([]PDFDescoberto, error)
	DownloadPDF(ctx context.Context, url string) (pdf []byte, err error)
}

type Rasterizer interface {
	Rasterize(ctx context.Context, pdf []byte) ([]PageImage, error)
}

// FilenameVigencia holds optional start/end dates parsed from a Documento filename (ADR 0028).
type FilenameVigencia struct {
	DataInicio    string // empty when no distinct start
	DataExpiracao string // empty when no end token
}

// FilenameDateParser extracts vigência hints from a Documento filename.
type FilenameDateParser interface {
	Parse(filename string) FilenameVigencia
}
