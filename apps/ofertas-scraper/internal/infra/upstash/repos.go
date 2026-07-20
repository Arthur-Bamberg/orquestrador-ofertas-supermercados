package upstash

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

const (
	keyMercados = "mercados"
	keyFontes   = "fontes"
)

func mercadoKey(id domain.MercadoID) string   { return "mercado:" + string(id) }
func fonteKey(id domain.FonteID) string       { return "fonte:" + string(id) }
func produtoKey(id domain.ProdutoID) string   { return "produto:" + string(id) }
func marcaKey(id domain.MarcaID) string       { return "marca:" + string(id) }
func documentoKey(id domain.DocumentoID) string {
	return "documento:" + string(id)
}
func produtoNormKey(norm string) string { return "produto:norm:" + norm }
func marcaNormKey(norm string) string   { return "marca:norm:" + norm }
func documentoIDKey(fonteID domain.FonteID, filename, dia string) string {
	return fmt.Sprintf("documento:id:%s:%s:%s", fonteID, filename, dia)
}
func documentoDiasKey(fonteID domain.FonteID, filename string) string {
	return fmt.Sprintf("documento:dias:%s:%s", fonteID, filename)
}
func ofertasDocKey(id domain.DocumentoID) string { return "ofertas:documento:" + string(id) }
func ofertasProdutoKey(id domain.ProdutoID) string {
	return "ofertas:produto:" + string(id)
}
func falhasDocKey(id domain.DocumentoID) string { return "falhas:documento:" + string(id) }
func usoExtratorKey(documentoID domain.DocumentoID, tentativa string) string {
	return fmt.Sprintf("uso-extrator:%s:%s", documentoID, tentativa)
}
func usoExtratorDocKey(id domain.DocumentoID) string {
	return "uso-extrator:documento:" + string(id)
}

type MercadoRepo struct{ c *Client }

func NewMercadoRepo(c *Client) *MercadoRepo { return &MercadoRepo{c: c} }

func (r *MercadoRepo) Save(ctx context.Context, m domain.Mercado) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := r.c.Set(ctx, mercadoKey(m.ID), string(b)); err != nil {
		return err
	}
	return r.c.SAdd(ctx, keyMercados, string(m.ID))
}

func (r *MercadoRepo) Get(ctx context.Context, id domain.MercadoID) (domain.Mercado, error) {
	raw, ok, err := r.c.Get(ctx, mercadoKey(id))
	if err != nil {
		return domain.Mercado{}, err
	}
	if !ok {
		return domain.Mercado{}, fmt.Errorf("mercado %s not found", id)
	}
	var m domain.Mercado
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return domain.Mercado{}, err
	}
	return m, nil
}

func (r *MercadoRepo) List(ctx context.Context) ([]domain.Mercado, error) {
	ids, err := r.c.SMembers(ctx, keyMercados)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Mercado, 0, len(ids))
	for _, id := range ids {
		m, err := r.Get(ctx, domain.MercadoID(id))
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

type FonteRepo struct{ c *Client }

func NewFonteRepo(c *Client) *FonteRepo { return &FonteRepo{c: c} }

func (r *FonteRepo) Save(ctx context.Context, f domain.Fonte) error {
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	if err := r.c.Set(ctx, fonteKey(f.ID), string(b)); err != nil {
		return err
	}
	return r.c.SAdd(ctx, keyFontes, string(f.ID))
}

func (r *FonteRepo) List(ctx context.Context) ([]domain.Fonte, error) {
	ids, err := r.c.SMembers(ctx, keyFontes)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Fonte, 0, len(ids))
	for _, id := range ids {
		raw, ok, err := r.c.Get(ctx, fonteKey(domain.FonteID(id)))
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		var f domain.Fonte
		if err := json.Unmarshal([]byte(raw), &f); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

type ProdutoRepo struct{ c *Client }

func NewProdutoRepo(c *Client) *ProdutoRepo { return &ProdutoRepo{c: c} }

func (r *ProdutoRepo) GetByNomeNorm(ctx context.Context, nomeNorm string) (domain.Produto, bool, error) {
	id, ok, err := r.c.Get(ctx, produtoNormKey(nomeNorm))
	if err != nil || !ok {
		return domain.Produto{}, ok, err
	}
	raw, ok, err := r.c.Get(ctx, produtoKey(domain.ProdutoID(id)))
	if err != nil || !ok {
		return domain.Produto{}, ok, err
	}
	var p domain.Produto
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return domain.Produto{}, false, err
	}
	return p, true, nil
}

func (r *ProdutoRepo) Save(ctx context.Context, p domain.Produto) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	if err := r.c.Set(ctx, produtoKey(p.ID), string(b)); err != nil {
		return err
	}
	return r.c.Set(ctx, produtoNormKey(p.NomeNorm), string(p.ID))
}

type MarcaRepo struct{ c *Client }

func NewMarcaRepo(c *Client) *MarcaRepo { return &MarcaRepo{c: c} }

func (r *MarcaRepo) GetByNomeNorm(ctx context.Context, nomeNorm string) (domain.Marca, bool, error) {
	id, ok, err := r.c.Get(ctx, marcaNormKey(nomeNorm))
	if err != nil || !ok {
		return domain.Marca{}, ok, err
	}
	raw, ok, err := r.c.Get(ctx, marcaKey(domain.MarcaID(id)))
	if err != nil || !ok {
		return domain.Marca{}, ok, err
	}
	var m domain.Marca
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return domain.Marca{}, false, err
	}
	return m, true, nil
}

func (r *MarcaRepo) Save(ctx context.Context, m domain.Marca) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := r.c.Set(ctx, marcaKey(m.ID), string(b)); err != nil {
		return err
	}
	return r.c.Set(ctx, marcaNormKey(m.NomeNorm), string(m.ID))
}

type DocumentoRepo struct{ c *Client }

func NewDocumentoRepo(c *Client) *DocumentoRepo { return &DocumentoRepo{c: c} }

func (r *DocumentoRepo) GetByIdentity(ctx context.Context, fonteID domain.FonteID, filename, dia string) (domain.Documento, bool, error) {
	id, ok, err := r.c.Get(ctx, documentoIDKey(fonteID, filename, dia))
	if err != nil || !ok {
		return domain.Documento{}, ok, err
	}
	raw, ok, err := r.c.Get(ctx, documentoKey(domain.DocumentoID(id)))
	if err != nil || !ok {
		return domain.Documento{}, ok, err
	}
	var d domain.Documento
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return domain.Documento{}, false, err
	}
	return d, true, nil
}

func (r *DocumentoRepo) Save(ctx context.Context, d domain.Documento) error {
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}
	if err := r.c.Set(ctx, documentoKey(d.ID), string(b)); err != nil {
		return err
	}
	if err := r.c.Set(ctx, documentoIDKey(d.FonteID, d.Filename, d.Dia), string(d.ID)); err != nil {
		return err
	}
	return r.c.SAdd(ctx, documentoDiasKey(d.FonteID, d.Filename), d.Dia)
}

func (r *DocumentoRepo) EarliestDia(ctx context.Context, fonteID domain.FonteID, filename string) (string, bool, error) {
	dias, err := r.c.SMembers(ctx, documentoDiasKey(fonteID, filename))
	if err != nil {
		return "", false, err
	}
	if len(dias) == 0 {
		return "", false, nil
	}
	earliest := dias[0]
	for _, d := range dias[1:] {
		if d < earliest {
			earliest = d
		}
	}
	return earliest, true, nil
}

type OfertaRepo struct{ c *Client }

func NewOfertaRepo(c *Client) *OfertaRepo { return &OfertaRepo{c: c} }

func (r *OfertaRepo) SaveAll(ctx context.Context, documentoID domain.DocumentoID, ofertas []domain.Oferta) error {
	if ofertas == nil {
		ofertas = []domain.Oferta{}
	}
	prev, err := r.ListByDocumento(ctx, documentoID)
	if err != nil {
		return err
	}
	prevProdutos := produtoIDsFromOfertas(prev)
	nextProdutos := produtoIDsFromOfertas(ofertas)
	docMember := string(documentoID)
	for pid := range prevProdutos {
		if _, ok := nextProdutos[pid]; ok {
			continue
		}
		if err := r.c.SRem(ctx, ofertasProdutoKey(pid), docMember); err != nil {
			return err
		}
	}
	for pid := range nextProdutos {
		if err := r.c.SAdd(ctx, ofertasProdutoKey(pid), docMember); err != nil {
			return err
		}
	}
	b, err := json.Marshal(ofertas)
	if err != nil {
		return err
	}
	return r.c.Set(ctx, ofertasDocKey(documentoID), string(b))
}

func (r *OfertaRepo) ListByDocumento(ctx context.Context, documentoID domain.DocumentoID) ([]domain.Oferta, error) {
	raw, ok, err := r.c.Get(ctx, ofertasDocKey(documentoID))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	var ofertas []domain.Oferta
	if err := json.Unmarshal([]byte(raw), &ofertas); err != nil {
		return nil, err
	}
	return ofertas, nil
}

func (r *OfertaRepo) ListDocumentoIDsByProduto(ctx context.Context, produtoID domain.ProdutoID) ([]domain.DocumentoID, error) {
	members, err := r.c.SMembers(ctx, ofertasProdutoKey(produtoID))
	if err != nil {
		return nil, err
	}
	ids := make([]domain.DocumentoID, 0, len(members))
	for _, m := range members {
		ids = append(ids, domain.DocumentoID(m))
	}
	return ids, nil
}

func produtoIDsFromOfertas(ofertas []domain.Oferta) map[domain.ProdutoID]struct{} {
	out := make(map[domain.ProdutoID]struct{})
	for _, o := range ofertas {
		if o.ProdutoID == "" {
			continue
		}
		out[o.ProdutoID] = struct{}{}
	}
	return out
}

type FalhaRepo struct{ c *Client }

func NewFalhaRepo(c *Client) *FalhaRepo { return &FalhaRepo{c: c} }

func (r *FalhaRepo) SaveAll(ctx context.Context, documentoID domain.DocumentoID, falhas []domain.FalhaExtracao) error {
	if falhas == nil {
		falhas = []domain.FalhaExtracao{}
	}
	b, err := json.Marshal(falhas)
	if err != nil {
		return err
	}
	return r.c.Set(ctx, falhasDocKey(documentoID), string(b))
}

type UsoExtratorRepo struct{ c *Client }

func NewUsoExtratorRepo(c *Client) *UsoExtratorRepo { return &UsoExtratorRepo{c: c} }

func (r *UsoExtratorRepo) Save(ctx context.Context, uso domain.UsoExtrator) error {
	if uso.DocumentoID == "" || uso.Tentativa == "" {
		return fmt.Errorf("uso-extrator: documentoId e tentativa são obrigatórios")
	}
	b, err := json.Marshal(uso)
	if err != nil {
		return err
	}
	if err := r.c.Set(ctx, usoExtratorKey(uso.DocumentoID, uso.Tentativa), string(b)); err != nil {
		return err
	}
	return r.c.SAdd(ctx, usoExtratorDocKey(uso.DocumentoID), uso.Tentativa)
}
