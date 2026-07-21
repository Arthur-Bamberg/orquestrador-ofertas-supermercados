package store

import (
	"context"
	"fmt"
)

type MercadoRepo struct{ catalog *Catalog }

func NewMercadoRepo(c *Client) *MercadoRepo { return &MercadoRepo{catalog: NewCatalog(c)} }

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

func NewFonteRepo(c *Client) *FonteRepo { return &FonteRepo{catalog: NewCatalog(c)} }

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

func NewProdutoRepo(c *Client) *ProdutoRepo { return &ProdutoRepo{catalog: NewCatalog(c)} }

func (r *ProdutoRepo) GetByNomeNorm(ctx context.Context, nomeNorm string) (Produto, bool, error) {
	return r.catalog.GetProdutoByNomeNorm(ctx, nomeNorm)
}

func (r *ProdutoRepo) Save(ctx context.Context, p Produto) error {
	return r.catalog.SaveProduto(ctx, p)
}

type MarcaRepo struct{ catalog *Catalog }

func NewMarcaRepo(c *Client) *MarcaRepo { return &MarcaRepo{catalog: NewCatalog(c)} }

func (r *MarcaRepo) GetByNomeNorm(ctx context.Context, nomeNorm string) (Marca, bool, error) {
	return r.catalog.GetMarcaByNomeNorm(ctx, nomeNorm)
}

func (r *MarcaRepo) Save(ctx context.Context, m Marca) error {
	return r.catalog.SaveMarca(ctx, m)
}

type DocumentoRepo struct{ catalog *Catalog }

func NewDocumentoRepo(c *Client) *DocumentoRepo { return &DocumentoRepo{catalog: NewCatalog(c)} }

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

func NewOfertaRepo(c *Client) *OfertaRepo { return &OfertaRepo{catalog: NewCatalog(c)} }

func (r *OfertaRepo) GetByUniq(ctx context.Context, chave string) (Oferta, bool, error) {
	return r.catalog.GetOfertaByUniq(ctx, chave)
}

func (r *OfertaRepo) SaveAll(ctx context.Context, documentoID DocumentoID, ofertas []Oferta) error {
	return r.catalog.SaveOfertasForDocumento(ctx, documentoID, ofertas)
}

func (r *OfertaRepo) ListByDocumento(ctx context.Context, documentoID DocumentoID) ([]Oferta, error) {
	return r.catalog.ListOfertasByDocumento(ctx, documentoID)
}

func (r *OfertaRepo) ListDocumentoIDsByProduto(ctx context.Context, produtoID ProdutoID) ([]DocumentoID, error) {
	return r.catalog.ListDocumentoIDsByProduto(ctx, produtoID)
}

type FalhaRepo struct{ catalog *Catalog }

func NewFalhaRepo(c *Client) *FalhaRepo { return &FalhaRepo{catalog: NewCatalog(c)} }

func (r *FalhaRepo) SaveAll(ctx context.Context, documentoID DocumentoID, falhas []FalhaExtracao) error {
	return r.catalog.SaveFalhasDocumento(ctx, documentoID, falhas)
}

type UsoExtratorRepo struct{ catalog *Catalog }

func NewUsoExtratorRepo(c *Client) *UsoExtratorRepo {
	return &UsoExtratorRepo{catalog: NewCatalog(c)}
}

func (r *UsoExtratorRepo) Save(ctx context.Context, uso UsoExtrator) error {
	return r.catalog.SaveUsoExtrator(ctx, uso)
}

type CotaRepo struct{ c *Client }

func NewCotaRepo(c *Client) *CotaRepo { return &CotaRepo{c: c} }

func extratorCotaKey(provider, dia string) string {
	return fmt.Sprintf("ofertas-scraper:extrator:%s:esgotado:%s", provider, dia)
}

func (r *CotaRepo) Esgotado(ctx context.Context, provider, dia string) (bool, error) {
	_, ok, err := r.c.Get(ctx, extratorCotaKey(provider, dia))
	return ok, err
}

func (r *CotaRepo) MarcarEsgotado(ctx context.Context, provider, dia string) error {
	return r.c.SetEX(ctx, extratorCotaKey(provider, dia), "1", 48*60*60)
}
