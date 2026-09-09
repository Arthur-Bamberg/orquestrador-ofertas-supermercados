package store

import (
	"context"
	"fmt"
)

type MercadoRepo struct{ catalog *Catalog }

func NewMercadoRepo(c *Catalog) *MercadoRepo { return &MercadoRepo{catalog: c} }

func (r *MercadoRepo) Save(ctx context.Context, m Mercado) error {
	return r.catalog.SaveMercado(ctx, m)
}

func (r *MercadoRepo) Get(ctx context.Context, id MercadoID) (Mercado, error) {
	m, ok, err := r.catalog.GetMercado(ctx, id)
	if err != nil {
		return Mercado{}, err
	}
	if !ok {
		return Mercado{}, fmt.Errorf("%w: mercado %s", ErrNotFound, id)
	}
	return m, nil
}

func (r *MercadoRepo) List(ctx context.Context) ([]Mercado, error) {
	return r.catalog.ListMercados(ctx)
}

type FonteRepo struct{ catalog *Catalog }

func NewFonteRepo(c *Catalog) *FonteRepo { return &FonteRepo{catalog: c} }

func (r *FonteRepo) Save(ctx context.Context, f Fonte) error {
	return r.catalog.SaveFonte(ctx, f)
}

func (r *FonteRepo) Get(ctx context.Context, id FonteID) (Fonte, bool, error) {
	return r.catalog.GetFonte(ctx, id)
}

func (r *FonteRepo) List(ctx context.Context) ([]Fonte, error) {
	return r.catalog.ListFontes(ctx)
}

type ProdutoRepo struct{ catalog *Catalog }

func NewProdutoRepo(c *Catalog) *ProdutoRepo { return &ProdutoRepo{catalog: c} }

func (r *ProdutoRepo) GetByNomeNorm(ctx context.Context, nomeNorm string) (Produto, bool, error) {
	return r.catalog.GetProdutoByNomeNorm(ctx, nomeNorm)
}

func (r *ProdutoRepo) Save(ctx context.Context, p Produto) error {
	return r.catalog.SaveProduto(ctx, p)
}

type MarcaRepo struct{ catalog *Catalog }

func NewMarcaRepo(c *Catalog) *MarcaRepo { return &MarcaRepo{catalog: c} }

func (r *MarcaRepo) GetByNomeNorm(ctx context.Context, nomeNorm string) (Marca, bool, error) {
	return r.catalog.GetMarcaByNomeNorm(ctx, nomeNorm)
}

func (r *MarcaRepo) Save(ctx context.Context, m Marca) error {
	return r.catalog.SaveMarca(ctx, m)
}

type DocumentoRepo struct{ catalog *Catalog }

func NewDocumentoRepo(c *Catalog) *DocumentoRepo { return &DocumentoRepo{catalog: c} }

func (r *DocumentoRepo) GetByIdentity(ctx context.Context, fonteID FonteID, filename, dia string) (Documento, bool, error) {
	return r.catalog.GetDocumentoByIdentity(ctx, fonteID, filename, dia)
}

func (r *DocumentoRepo) Get(ctx context.Context, id DocumentoID) (Documento, bool, error) {
	return r.catalog.GetDocumento(ctx, id)
}

func (r *DocumentoRepo) Save(ctx context.Context, d Documento) error {
	return r.catalog.SaveDocumento(ctx, d)
}

func (r *DocumentoRepo) EarliestDia(ctx context.Context, fonteID FonteID, filename string) (string, bool, error) {
	return r.catalog.EarliestDia(ctx, fonteID, filename)
}

func (r *DocumentoRepo) ListDias(ctx context.Context, fonteID FonteID, filename string) ([]string, error) {
	return r.catalog.ListDias(ctx, fonteID, filename)
}

type OfertaRepo struct{ catalog *Catalog }

func NewOfertaRepo(c *Catalog) *OfertaRepo { return &OfertaRepo{catalog: c} }

func (r *OfertaRepo) GetByUniq(ctx context.Context, chave string) (Oferta, bool, error) {
	return r.catalog.GetOfertaByUniq(ctx, chave)
}

func (r *OfertaRepo) SaveAll(ctx context.Context, documentoID DocumentoID, ofertas []Oferta) error {
	return r.catalog.SaveOfertasForDocumento(ctx, documentoID, ofertas)
}

func (r *OfertaRepo) SaveAllForColeta(ctx context.Context, coletaID ColetaID, ofertas []Oferta) error {
	return r.catalog.SaveOfertasForColeta(ctx, coletaID, ofertas)
}

func (r *OfertaRepo) ListByDocumento(ctx context.Context, documentoID DocumentoID) ([]Oferta, error) {
	return r.catalog.ListOfertasByDocumento(ctx, documentoID)
}

func (r *OfertaRepo) ListByColeta(ctx context.Context, coletaID ColetaID) ([]Oferta, error) {
	return r.catalog.ListOfertasByColeta(ctx, coletaID)
}

func (r *OfertaRepo) ListDocumentoIDsByProduto(ctx context.Context, produtoID ProdutoID) ([]DocumentoID, error) {
	return r.catalog.ListDocumentoIDsByProduto(ctx, produtoID)
}

type ColetaRepo struct{ catalog *Catalog }

func NewColetaRepo(c *Catalog) *ColetaRepo { return &ColetaRepo{catalog: c} }

func (r *ColetaRepo) Save(ctx context.Context, c Coleta) error {
	return r.catalog.SaveColeta(ctx, c)
}

func (r *ColetaRepo) Get(ctx context.Context, id ColetaID) (Coleta, bool, error) {
	return r.catalog.GetColeta(ctx, id)
}

func (r *ColetaRepo) GetByIdentity(ctx context.Context, termo string, mercadoID MercadoID, dia string) (Coleta, bool, error) {
	return r.catalog.GetColetaByIdentity(ctx, termo, mercadoID, dia)
}

type FalhaRepo struct{ catalog *Catalog }

func NewFalhaRepo(c *Catalog) *FalhaRepo { return &FalhaRepo{catalog: c} }

func (r *FalhaRepo) SaveAll(ctx context.Context, documentoID DocumentoID, falhas []FalhaExtracao) error {
	return r.catalog.SaveFalhasDocumento(ctx, documentoID, falhas)
}

type UsoExtratorRepo struct{ catalog *Catalog }

func NewUsoExtratorRepo(c *Catalog) *UsoExtratorRepo {
	return &UsoExtratorRepo{catalog: c}
}

func (r *UsoExtratorRepo) Save(ctx context.Context, uso UsoExtrator) error {
	return r.catalog.SaveUsoExtrator(ctx, uso)
}

type CotaRepo struct{ catalog *Catalog }

func NewCotaRepo(c *Catalog) *CotaRepo { return &CotaRepo{catalog: c} }

func (r *CotaRepo) Esgotado(ctx context.Context, provider, dia string) (bool, error) {
	return r.catalog.Esgotado(ctx, provider, dia)
}

func (r *CotaRepo) MarcarEsgotado(ctx context.Context, provider, dia string) error {
	return r.catalog.MarcarEsgotado(ctx, provider, dia)
}
