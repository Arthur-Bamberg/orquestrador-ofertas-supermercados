package domain

import (
	"context"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

type MercadoID = store.MercadoID
type FonteID = store.FonteID
type ProdutoID = store.ProdutoID
type MarcaID = store.MarcaID
type DocumentoID = store.DocumentoID
type OfertaID = store.OfertaID

type Mercado = store.Mercado

type Fonte = store.Fonte

type TipoFonte = store.TipoFonte

const (
	TipoEncarte = store.TipoEncarte
	TipoSite    = store.TipoSite
)

type Produto = store.Produto

type Marca = store.Marca

type Documento = store.Documento

// OrigemData records how a vigência date was obtained (ADR 0028).
type OrigemData = store.OrigemData

const (
	OrigemExtrator           = store.OrigemExtrator
	OrigemFilename           = store.OrigemFilename
	OrigemPrimeiraDescoberta = store.OrigemPrimeiraDescoberta
)

// Oferta is the unique catalog price observation (ADR 0036). DocumentoID is optional
// legacy/debug (first association); ownership is N∶1 via OfertaRepository associations.
type Oferta = store.Oferta

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
	Get(ctx context.Context, id DocumentoID) (Documento, bool, error)
	Save(ctx context.Context, d Documento) error
	// EarliestDia is the smallest discovery day for Fonte+filename (any Documento state).
	EarliestDia(ctx context.Context, fonteID FonteID, filename string) (dia string, ok bool, err error)
	// ListDias returns discovery days for Fonte+filename (any order).
	ListDias(ctx context.Context, fonteID FonteID, filename string) ([]string, error)
}

type OfertaRepository interface {
	// SaveAll replaces Documento↔Oferta associations, upserts catalog entities by uniqueness
	// key, and deletes Ofertas that become unreferenced (ADR 0036).
	SaveAll(ctx context.Context, documentoID DocumentoID, ofertas []Oferta) error
	ListByDocumento(ctx context.Context, documentoID DocumentoID) ([]Oferta, error)
	GetByUniq(ctx context.Context, chave string) (Oferta, bool, error)
	// ListDocumentoIDsByProduto returns Documentos that currently associate Ofertas of this Produto (ADR 0026 / 0038).
	ListDocumentoIDsByProduto(ctx context.Context, produtoID ProdutoID) ([]DocumentoID, error)
}

type FalhaExtracaoRepository interface {
	SaveAll(ctx context.Context, documentoID DocumentoID, falhas []FalhaExtracao) error
}

type Medida = store.Medida

const (
	MedidaG       = store.MedidaG
	MedidaML      = store.MedidaML
	MedidaUnidade = store.MedidaUnidade
)

type Promocao = store.Promocao
type Comparativo = store.Comparativo
type CandidatoOferta = store.CandidatoOferta
type FalhaExtracao = store.FalhaExtracao
type PageImage = store.PageImage

// UsoExtratorPagina is per-page token usage inside one Extract call (ADR 0031/0033).
type UsoExtratorPagina = store.UsoExtratorPagina

// UsoExtrator records Extrator token consumption for one processing tentativa (ADR 0033/0037).
type UsoExtrator = store.UsoExtrator

// ExtratorCotaStore persists daily adapter quota exhaustion (ADR 0037 / 0038).
type ExtratorCotaStore interface {
	Esgotado(ctx context.Context, provider, dia string) (bool, error)
	MarcarEsgotado(ctx context.Context, provider, dia string) error
}

type Extrator interface {
	// Extract returns candidatos, raw JSON, and optional Uso (nil when the adapter has no usage).
	Extract(ctx context.Context, images []PageImage) (candidatos []CandidatoOferta, raw []byte, uso *UsoExtrator, err error)
}

type UsoExtratorRepository interface {
	Save(ctx context.Context, uso UsoExtrator) error
}

// PDFDescoberto is a PDF link found on a Fonte page (ADR 0020).
type PDFDescoberto = store.PDFDescoberto

type ArtefatoStore interface {
	// AttemptPath is the directory for a Documento tentativa (relative or absolute under the store root).
	AttemptPath(doc Documento, tentativa string) string
	SavePDF(ctx context.Context, doc Documento, tentativa string, pdf []byte) error
	SaveImages(ctx context.Context, doc Documento, tentativa string, images []PageImage) error
	SaveRawExtrator(ctx context.Context, doc Documento, tentativa string, raw []byte) error
	SaveUsoExtrator(ctx context.Context, doc Documento, tentativa string, uso UsoExtrator) error
	SaveValidated(ctx context.Context, doc Documento, tentativa string, ofertas []Oferta, falhas []FalhaExtracao) error
	// SaveConteudoIdentico records the pointer to the prior Documento (ADR 0036 shortcut).
	SaveConteudoIdentico(ctx context.Context, doc Documento, tentativa string, priorID DocumentoID) error
}

type FonteClient interface {
	DiscoverPDFs(ctx context.Context, fonte Fonte) ([]PDFDescoberto, error)
	DownloadPDF(ctx context.Context, url string) (pdf []byte, err error)
}

type Rasterizer interface {
	Rasterize(ctx context.Context, pdf []byte) ([]PageImage, error)
}

// FilenameVigencia holds optional start/end dates parsed from a Documento filename (ADR 0028).
type FilenameVigencia = store.FilenameVigencia

// FilenameDateParser extracts vigência hints from a Documento filename.
type FilenameDateParser interface {
	Parse(filename string) FilenameVigencia
}
