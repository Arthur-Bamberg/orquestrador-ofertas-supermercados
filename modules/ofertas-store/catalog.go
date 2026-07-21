package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrDeleteBlocked = errors.New("delete blocked")
	ErrConflict      = errors.New("conflict")
	ErrInvalid       = errors.New("invalid")
)

const (
	keyMercados   = "mercados"
	keyFontes     = "fontes"
	keyProdutos   = "produtos"
	keyMarcas     = "marcas"
	keyDocumentos = "documentos"
	keyOfertas    = "ofertas"
	keyUsos       = "usos-extrator"
)

func mercadoKey(id MercadoID) string     { return "mercado:" + string(id) }
func fonteKey(id FonteID) string         { return "fonte:" + string(id) }
func produtoKey(id ProdutoID) string     { return "produto:" + string(id) }
func marcaKey(id MarcaID) string         { return "marca:" + string(id) }
func documentoKey(id DocumentoID) string { return "documento:" + string(id) }
func produtoNormKey(norm string) string  { return "produto:norm:" + norm }
func marcaNormKey(norm string) string    { return "marca:norm:" + norm }
func documentoIDKey(fonteID FonteID, filename, dia string) string {
	return fmt.Sprintf("documento:id:%s:%s:%s", fonteID, filename, dia)
}
func documentoDiasKey(fonteID FonteID, filename string) string {
	return fmt.Sprintf("documento:dias:%s:%s", fonteID, filename)
}
func ofertasDocKey(id DocumentoID) string    { return "ofertas:documento:" + string(id) }
func ofertaKey(id OfertaID) string           { return "oferta:" + string(id) }
func ofertaUniqKey(chave string) string      { return "oferta:uniq:" + chave }
func ofertaDocumentosKey(id OfertaID) string { return "oferta:documentos:" + string(id) }
func ofertasProdutoKey(id ProdutoID) string  { return "ofertas:produto:" + string(id) }
func falhasDocKey(id DocumentoID) string     { return "falhas:documento:" + string(id) }
func usoExtratorKey(documentoID DocumentoID, tentativa string) string {
	return fmt.Sprintf("uso-extrator:%s:%s", documentoID, tentativa)
}
func usoExtratorDocKey(id DocumentoID) string { return "uso-extrator:documento:" + string(id) }

type Catalog struct {
	c *Client
}

func NewCatalog(c *Client) *Catalog {
	return &Catalog{c: c}
}

func (s *Catalog) SaveMercado(ctx context.Context, m Mercado) error {
	if m.ID == "" {
		return fmt.Errorf("%w: mercado id obrigatório", ErrInvalid)
	}
	if err := s.saveJSON(ctx, mercadoKey(m.ID), m); err != nil {
		return err
	}
	return s.c.SAdd(ctx, keyMercados, string(m.ID))
}

func (s *Catalog) GetMercado(ctx context.Context, id MercadoID) (Mercado, bool, error) {
	var m Mercado
	ok, err := s.getJSON(ctx, mercadoKey(id), &m)
	return m, ok, err
}

func (s *Catalog) ListMercados(ctx context.Context) ([]Mercado, error) {
	ids, err := s.idsFromSetOrKeys(ctx, keyMercados, "mercado:*", "mercado:")
	if err != nil {
		return nil, err
	}
	out := make([]Mercado, 0, len(ids))
	for _, id := range ids {
		m, ok, err := s.GetMercado(ctx, MercadoID(id))
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, m)
		}
	}
	return out, nil
}

func (s *Catalog) DeleteMercado(ctx context.Context, id MercadoID) error {
	fontes, err := s.ListFontes(ctx)
	if err != nil {
		return err
	}
	for _, f := range fontes {
		if f.MercadoID == id {
			return fmt.Errorf("%w: mercado %s tem fontes", ErrDeleteBlocked, id)
		}
	}
	if err := s.c.Del(ctx, mercadoKey(id)); err != nil {
		return err
	}
	return s.c.SRem(ctx, keyMercados, string(id))
}

func (s *Catalog) SaveFonte(ctx context.Context, f Fonte) error {
	if f.ID == "" {
		return fmt.Errorf("%w: fonte id obrigatório", ErrInvalid)
	}
	if err := s.saveJSON(ctx, fonteKey(f.ID), f); err != nil {
		return err
	}
	return s.c.SAdd(ctx, keyFontes, string(f.ID))
}

func (s *Catalog) GetFonte(ctx context.Context, id FonteID) (Fonte, bool, error) {
	var f Fonte
	ok, err := s.getJSON(ctx, fonteKey(id), &f)
	return f, ok, err
}

func (s *Catalog) ListFontes(ctx context.Context) ([]Fonte, error) {
	ids, err := s.idsFromSetOrKeys(ctx, keyFontes, "fonte:*", "fonte:")
	if err != nil {
		return nil, err
	}
	out := make([]Fonte, 0, len(ids))
	for _, id := range ids {
		f, ok, err := s.GetFonte(ctx, FonteID(id))
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, f)
		}
	}
	return out, nil
}

func (s *Catalog) DeleteFonte(ctx context.Context, id FonteID) error {
	documentos, err := s.ListDocumentos(ctx)
	if err != nil {
		return err
	}
	for _, d := range documentos {
		if d.FonteID == id {
			return fmt.Errorf("%w: fonte %s tem documentos", ErrDeleteBlocked, id)
		}
	}
	if err := s.c.Del(ctx, fonteKey(id)); err != nil {
		return err
	}
	return s.c.SRem(ctx, keyFontes, string(id))
}

func (s *Catalog) SaveProduto(ctx context.Context, p Produto) error {
	if p.ID == "" {
		return fmt.Errorf("%w: produto id obrigatório", ErrInvalid)
	}
	if old, ok, err := s.GetProduto(ctx, p.ID); err != nil {
		return err
	} else if ok && old.NomeNorm != "" && old.NomeNorm != p.NomeNorm {
		if err := s.c.Del(ctx, produtoNormKey(old.NomeNorm)); err != nil {
			return err
		}
	}
	if err := s.saveJSON(ctx, produtoKey(p.ID), p); err != nil {
		return err
	}
	if p.NomeNorm != "" {
		if err := s.c.Set(ctx, produtoNormKey(p.NomeNorm), string(p.ID)); err != nil {
			return err
		}
	}
	return s.c.SAdd(ctx, keyProdutos, string(p.ID))
}

func (s *Catalog) GetProduto(ctx context.Context, id ProdutoID) (Produto, bool, error) {
	var p Produto
	ok, err := s.getJSON(ctx, produtoKey(id), &p)
	return p, ok, err
}

func (s *Catalog) GetProdutoByNomeNorm(ctx context.Context, nomeNorm string) (Produto, bool, error) {
	id, ok, err := s.c.Get(ctx, produtoNormKey(nomeNorm))
	if err != nil || !ok {
		return Produto{}, ok, err
	}
	return s.GetProduto(ctx, ProdutoID(id))
}

func (s *Catalog) ListProdutos(ctx context.Context) ([]Produto, error) {
	ids, err := s.idsFromSetOrKeys(ctx, keyProdutos, "produto:*", "produto:")
	if err != nil {
		return nil, err
	}
	out := make([]Produto, 0, len(ids))
	for _, id := range ids {
		if strings.HasPrefix(id, "norm:") {
			continue
		}
		p, ok, err := s.GetProduto(ctx, ProdutoID(id))
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *Catalog) DeleteProduto(ctx context.Context, id ProdutoID) error {
	ofertas, err := s.ListOfertas(ctx)
	if err != nil {
		return err
	}
	for _, o := range ofertas {
		if o.ProdutoID == id {
			return fmt.Errorf("%w: produto %s referenciado por ofertas", ErrDeleteBlocked, id)
		}
	}
	p, ok, err := s.GetProduto(ctx, id)
	if err != nil {
		return err
	}
	keys := []string{produtoKey(id), ofertasProdutoKey(id)}
	if ok && p.NomeNorm != "" {
		keys = append(keys, produtoNormKey(p.NomeNorm))
	}
	if err := s.c.Del(ctx, keys...); err != nil {
		return err
	}
	return s.c.SRem(ctx, keyProdutos, string(id))
}

func (s *Catalog) SaveMarca(ctx context.Context, m Marca) error {
	if m.ID == "" {
		return fmt.Errorf("%w: marca id obrigatório", ErrInvalid)
	}
	if old, ok, err := s.GetMarca(ctx, m.ID); err != nil {
		return err
	} else if ok && old.NomeNorm != "" && old.NomeNorm != m.NomeNorm {
		if err := s.c.Del(ctx, marcaNormKey(old.NomeNorm)); err != nil {
			return err
		}
	}
	if err := s.saveJSON(ctx, marcaKey(m.ID), m); err != nil {
		return err
	}
	if m.NomeNorm != "" {
		if err := s.c.Set(ctx, marcaNormKey(m.NomeNorm), string(m.ID)); err != nil {
			return err
		}
	}
	return s.c.SAdd(ctx, keyMarcas, string(m.ID))
}

func (s *Catalog) GetMarca(ctx context.Context, id MarcaID) (Marca, bool, error) {
	var m Marca
	ok, err := s.getJSON(ctx, marcaKey(id), &m)
	return m, ok, err
}

func (s *Catalog) GetMarcaByNomeNorm(ctx context.Context, nomeNorm string) (Marca, bool, error) {
	id, ok, err := s.c.Get(ctx, marcaNormKey(nomeNorm))
	if err != nil || !ok {
		return Marca{}, ok, err
	}
	return s.GetMarca(ctx, MarcaID(id))
}

func (s *Catalog) ListMarcas(ctx context.Context) ([]Marca, error) {
	ids, err := s.idsFromSetOrKeys(ctx, keyMarcas, "marca:*", "marca:")
	if err != nil {
		return nil, err
	}
	out := make([]Marca, 0, len(ids))
	for _, id := range ids {
		if strings.HasPrefix(id, "norm:") {
			continue
		}
		m, ok, err := s.GetMarca(ctx, MarcaID(id))
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, m)
		}
	}
	return out, nil
}

func (s *Catalog) DeleteMarca(ctx context.Context, id MarcaID) error {
	ofertas, err := s.ListOfertas(ctx)
	if err != nil {
		return err
	}
	for _, o := range ofertas {
		if o.MarcaID != nil && *o.MarcaID == id {
			return fmt.Errorf("%w: marca %s referenciada por ofertas", ErrDeleteBlocked, id)
		}
	}
	m, ok, err := s.GetMarca(ctx, id)
	if err != nil {
		return err
	}
	keys := []string{marcaKey(id)}
	if ok && m.NomeNorm != "" {
		keys = append(keys, marcaNormKey(m.NomeNorm))
	}
	if err := s.c.Del(ctx, keys...); err != nil {
		return err
	}
	return s.c.SRem(ctx, keyMarcas, string(id))
}

func (s *Catalog) SaveDocumento(ctx context.Context, d Documento) error {
	if d.ID == "" {
		return fmt.Errorf("%w: documento id obrigatório", ErrInvalid)
	}
	if err := s.saveJSON(ctx, documentoKey(d.ID), d); err != nil {
		return err
	}
	if err := s.c.Set(ctx, documentoIDKey(d.FonteID, d.Filename, d.Dia), string(d.ID)); err != nil {
		return err
	}
	if err := s.c.SAdd(ctx, documentoDiasKey(d.FonteID, d.Filename), d.Dia); err != nil {
		return err
	}
	return s.c.SAdd(ctx, keyDocumentos, string(d.ID))
}

func (s *Catalog) GetDocumento(ctx context.Context, id DocumentoID) (Documento, bool, error) {
	var d Documento
	ok, err := s.getJSON(ctx, documentoKey(id), &d)
	return d, ok, err
}

func (s *Catalog) GetDocumentoByIdentity(ctx context.Context, fonteID FonteID, filename, dia string) (Documento, bool, error) {
	id, ok, err := s.c.Get(ctx, documentoIDKey(fonteID, filename, dia))
	if err != nil || !ok {
		return Documento{}, ok, err
	}
	return s.GetDocumento(ctx, DocumentoID(id))
}

func (s *Catalog) ListDocumentos(ctx context.Context) ([]Documento, error) {
	ids, err := s.idsFromSetOrKeys(ctx, keyDocumentos, "documento:*", "documento:")
	if err != nil {
		return nil, err
	}
	out := make([]Documento, 0, len(ids))
	for _, id := range ids {
		if strings.HasPrefix(id, "id:") || strings.HasPrefix(id, "dias:") {
			continue
		}
		d, ok, err := s.GetDocumento(ctx, DocumentoID(id))
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, d)
		}
	}
	return out, nil
}

func (s *Catalog) DeleteDocumento(ctx context.Context, id DocumentoID) error {
	d, ok, err := s.GetDocumento(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: documento %s", ErrNotFound, id)
	}
	if err := s.SaveOfertasForDocumento(ctx, id, nil); err != nil {
		return err
	}
	if err := s.c.Del(ctx, falhasDocKey(id), ofertasDocKey(id), documentoKey(id), documentoIDKey(d.FonteID, d.Filename, d.Dia)); err != nil {
		return err
	}
	if err := s.c.SRem(ctx, documentoDiasKey(d.FonteID, d.Filename), d.Dia); err != nil {
		return err
	}
	return s.c.SRem(ctx, keyDocumentos, string(id))
}

func (s *Catalog) EarliestDia(ctx context.Context, fonteID FonteID, filename string) (string, bool, error) {
	dias, err := s.ListDias(ctx, fonteID, filename)
	if err != nil {
		return "", false, err
	}
	if len(dias) == 0 {
		return "", false, nil
	}
	earliest := dias[0]
	for _, dia := range dias[1:] {
		if dia < earliest {
			earliest = dia
		}
	}
	return earliest, true, nil
}

func (s *Catalog) ListDias(ctx context.Context, fonteID FonteID, filename string) ([]string, error) {
	return s.c.SMembers(ctx, documentoDiasKey(fonteID, filename))
}

func (s *Catalog) SaveOferta(ctx context.Context, o Oferta, documentoIDs []DocumentoID) error {
	if o.ID == "" {
		return fmt.Errorf("%w: oferta id obrigatório", ErrInvalid)
	}
	if len(documentoIDs) == 0 {
		return fmt.Errorf("%w: oferta exige ao menos um documento", ErrInvalid)
	}
	chave := ChaveUnicaOferta(o)
	if id, ok, err := s.c.Get(ctx, ofertaUniqKey(chave)); err != nil {
		return err
	} else if ok && OfertaID(id) != o.ID {
		return fmt.Errorf("%w: chave unica de oferta já existe", ErrConflict)
	}

	old, hadOld, err := s.GetOferta(ctx, o.ID)
	if err != nil {
		return err
	}
	oldChave := ""
	if hadOld {
		oldChave = ChaveUnicaOferta(old)
		if oldChave != chave {
			if err := s.c.Del(ctx, ofertaUniqKey(oldChave)); err != nil {
				return err
			}
		}
	}
	if o.DocumentoID == "" {
		o.DocumentoID = documentoIDs[0]
	}
	if err := s.saveOfertaEntity(ctx, o); err != nil {
		return err
	}

	prevMembers, err := s.c.SMembers(ctx, ofertaDocumentosKey(o.ID))
	if err != nil {
		return err
	}
	next := map[DocumentoID]struct{}{}
	for _, id := range documentoIDs {
		next[id] = struct{}{}
	}
	affected := map[DocumentoID]struct{}{}
	for _, raw := range prevMembers {
		docID := DocumentoID(raw)
		if _, keep := next[docID]; !keep {
			if err := s.removeOfertaFromDocumento(ctx, docID, o.ID); err != nil {
				return err
			}
			if err := s.c.SRem(ctx, ofertaDocumentosKey(o.ID), raw); err != nil {
				return err
			}
			affected[docID] = struct{}{}
		}
	}
	for docID := range next {
		if err := s.addOfertaToDocumento(ctx, docID, o.ID); err != nil {
			return err
		}
		if err := s.c.SAdd(ctx, ofertaDocumentosKey(o.ID), string(docID)); err != nil {
			return err
		}
		affected[docID] = struct{}{}
	}
	for docID := range affected {
		if hadOld {
			if err := s.refreshProdutoIndexForDocumento(ctx, docID, old.ProdutoID, o.ProdutoID); err != nil {
				return err
			}
		} else if err := s.refreshProdutoIndexForDocumento(ctx, docID, o.ProdutoID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Catalog) GetOferta(ctx context.Context, id OfertaID) (Oferta, bool, error) {
	return s.getOferta(ctx, id)
}

func (s *Catalog) GetOfertaByUniq(ctx context.Context, chave string) (Oferta, bool, error) {
	id, ok, err := s.c.Get(ctx, ofertaUniqKey(chave))
	if err != nil || !ok {
		return Oferta{}, ok, err
	}
	return s.getOferta(ctx, OfertaID(id))
}

func (s *Catalog) ListOfertas(ctx context.Context) ([]Oferta, error) {
	ids, err := s.idsFromSetOrKeys(ctx, keyOfertas, "oferta:*", "oferta:")
	if err != nil {
		return nil, err
	}
	out := make([]Oferta, 0, len(ids))
	for _, id := range ids {
		if strings.HasPrefix(id, "uniq:") || strings.HasPrefix(id, "documentos:") {
			continue
		}
		o, ok, err := s.GetOferta(ctx, OfertaID(id))
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, o)
		}
	}
	return out, nil
}

func (s *Catalog) DeleteOferta(ctx context.Context, id OfertaID) error {
	o, ok, err := s.GetOferta(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: oferta %s", ErrNotFound, id)
	}
	docIDs, err := s.c.SMembers(ctx, ofertaDocumentosKey(id))
	if err != nil {
		return err
	}
	for _, raw := range docIDs {
		docID := DocumentoID(raw)
		if err := s.removeOfertaFromDocumento(ctx, docID, id); err != nil {
			return err
		}
		if err := s.refreshProdutoIndexForDocumento(ctx, docID, o.ProdutoID); err != nil {
			return err
		}
	}
	if err := s.c.Del(ctx, ofertaKey(id), ofertaDocumentosKey(id), ofertaUniqKey(ChaveUnicaOferta(o))); err != nil {
		return err
	}
	return s.c.SRem(ctx, keyOfertas, string(id))
}

func (s *Catalog) SaveOfertasForDocumento(ctx context.Context, documentoID DocumentoID, ofertas []Oferta) error {
	if ofertas == nil {
		ofertas = []Oferta{}
	}
	prevIDs, err := s.listAssocIDs(ctx, documentoID)
	if err != nil {
		return err
	}
	prevSet := map[OfertaID]struct{}{}
	for _, id := range prevIDs {
		prevSet[id] = struct{}{}
	}

	nextIDs := make([]OfertaID, 0, len(ofertas))
	nextSet := map[OfertaID]struct{}{}
	nextProdutos := map[ProdutoID]struct{}{}

	for _, o := range ofertas {
		chave := ChaveUnicaOferta(o)
		existing, ok, err := s.GetOfertaByUniq(ctx, chave)
		if err != nil {
			return err
		}
		if ok {
			o = existing
		} else {
			if o.ID == "" {
				return fmt.Errorf("%w: oferta id obrigatório para criar entidade", ErrInvalid)
			}
			if err := s.saveOfertaEntity(ctx, o); err != nil {
				return err
			}
		}
		nextIDs = append(nextIDs, o.ID)
		nextSet[o.ID] = struct{}{}
		if o.ProdutoID != "" {
			nextProdutos[o.ProdutoID] = struct{}{}
		}
		if err := s.c.SAdd(ctx, ofertaDocumentosKey(o.ID), string(documentoID)); err != nil {
			return err
		}
	}

	if err := s.saveJSON(ctx, ofertasDocKey(documentoID), nextIDs); err != nil {
		return err
	}

	prevOfertas, err := s.loadOfertas(ctx, prevIDs)
	if err != nil {
		return err
	}
	prevProdutos := produtoIDsFromOfertas(prevOfertas)
	for pid := range prevProdutos {
		if _, ok := nextProdutos[pid]; ok {
			continue
		}
		if err := s.c.SRem(ctx, ofertasProdutoKey(pid), string(documentoID)); err != nil {
			return err
		}
	}
	for pid := range nextProdutos {
		if err := s.c.SAdd(ctx, ofertasProdutoKey(pid), string(documentoID)); err != nil {
			return err
		}
	}

	for id := range prevSet {
		if _, ok := nextSet[id]; ok {
			continue
		}
		if err := s.c.SRem(ctx, ofertaDocumentosKey(id), string(documentoID)); err != nil {
			return err
		}
		members, err := s.c.SMembers(ctx, ofertaDocumentosKey(id))
		if err != nil {
			return err
		}
		if len(members) > 0 {
			continue
		}
		o, ok, err := s.getOferta(ctx, id)
		if err != nil {
			return err
		}
		if ok {
			if err := s.c.Del(ctx, ofertaUniqKey(ChaveUnicaOferta(o))); err != nil {
				return err
			}
		}
		if err := s.c.Del(ctx, ofertaKey(id), ofertaDocumentosKey(id)); err != nil {
			return err
		}
		if err := s.c.SRem(ctx, keyOfertas, string(id)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Catalog) ListOfertasByDocumento(ctx context.Context, documentoID DocumentoID) ([]Oferta, error) {
	ids, err := s.listAssocIDs(ctx, documentoID)
	if err != nil {
		return nil, err
	}
	return s.loadOfertas(ctx, ids)
}

func (s *Catalog) ListDocumentoIDsByProduto(ctx context.Context, produtoID ProdutoID) ([]DocumentoID, error) {
	members, err := s.c.SMembers(ctx, ofertasProdutoKey(produtoID))
	if err != nil {
		return nil, err
	}
	out := make([]DocumentoID, 0, len(members))
	for _, member := range members {
		out = append(out, DocumentoID(member))
	}
	return out, nil
}

func (s *Catalog) SaveFalhasDocumento(ctx context.Context, documentoID DocumentoID, falhas []FalhaExtracao) error {
	if falhas == nil {
		falhas = []FalhaExtracao{}
	}
	return s.saveJSON(ctx, falhasDocKey(documentoID), falhas)
}

func (s *Catalog) GetFalhasDocumento(ctx context.Context, documentoID DocumentoID) ([]FalhaExtracao, error) {
	var falhas []FalhaExtracao
	ok, err := s.getJSON(ctx, falhasDocKey(documentoID), &falhas)
	if err != nil || !ok {
		return nil, err
	}
	return falhas, nil
}

func (s *Catalog) DeleteFalhasDocumento(ctx context.Context, documentoID DocumentoID) error {
	return s.c.Del(ctx, falhasDocKey(documentoID))
}

func (s *Catalog) SaveUsoExtrator(ctx context.Context, uso UsoExtrator) error {
	if uso.DocumentoID == "" || uso.Tentativa == "" {
		return fmt.Errorf("%w: uso-extrator documentoId e tentativa obrigatórios", ErrInvalid)
	}
	key := usoExtratorKey(uso.DocumentoID, uso.Tentativa)
	if err := s.saveJSON(ctx, key, uso); err != nil {
		return err
	}
	if err := s.c.SAdd(ctx, usoExtratorDocKey(uso.DocumentoID), uso.Tentativa); err != nil {
		return err
	}
	return s.c.SAdd(ctx, keyUsos, key)
}

func (s *Catalog) GetUsoExtrator(ctx context.Context, documentoID DocumentoID, tentativa string) (UsoExtrator, bool, error) {
	var uso UsoExtrator
	ok, err := s.getJSON(ctx, usoExtratorKey(documentoID, tentativa), &uso)
	return uso, ok, err
}

func (s *Catalog) ListUsosExtrator(ctx context.Context, documentoID DocumentoID) ([]UsoExtrator, error) {
	var keys []string
	var err error
	if documentoID != "" {
		tentativas, err := s.c.SMembers(ctx, usoExtratorDocKey(documentoID))
		if err != nil {
			return nil, err
		}
		for _, tentativa := range tentativas {
			keys = append(keys, usoExtratorKey(documentoID, tentativa))
		}
	} else {
		keys, err = s.c.SMembers(ctx, keyUsos)
		if err != nil {
			return nil, err
		}
		if len(keys) == 0 {
			keys, err = s.c.Keys(ctx, "uso-extrator:*:*")
			if err != nil {
				return nil, err
			}
		}
	}
	sort.Strings(keys)
	out := make([]UsoExtrator, 0, len(keys))
	for _, key := range keys {
		if strings.HasPrefix(key, "uso-extrator:documento:") {
			continue
		}
		var uso UsoExtrator
		ok, err := s.getJSON(ctx, key, &uso)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, uso)
		}
	}
	return out, nil
}

func (s *Catalog) DeleteUsoExtrator(ctx context.Context, documentoID DocumentoID, tentativa string) error {
	key := usoExtratorKey(documentoID, tentativa)
	if err := s.c.Del(ctx, key); err != nil {
		return err
	}
	if err := s.c.SRem(ctx, usoExtratorDocKey(documentoID), tentativa); err != nil {
		return err
	}
	return s.c.SRem(ctx, keyUsos, key)
}

func (s *Catalog) getOferta(ctx context.Context, id OfertaID) (Oferta, bool, error) {
	var o Oferta
	ok, err := s.getJSON(ctx, ofertaKey(id), &o)
	return o, ok, err
}

func (s *Catalog) saveOfertaEntity(ctx context.Context, o Oferta) error {
	if err := s.saveJSON(ctx, ofertaKey(o.ID), o); err != nil {
		return err
	}
	if err := s.c.Set(ctx, ofertaUniqKey(ChaveUnicaOferta(o)), string(o.ID)); err != nil {
		return err
	}
	return s.c.SAdd(ctx, keyOfertas, string(o.ID))
}

func (s *Catalog) listAssocIDs(ctx context.Context, documentoID DocumentoID) ([]OfertaID, error) {
	raw, ok, err := s.c.Get(ctx, ofertasDocKey(documentoID))
	if err != nil {
		return nil, err
	}
	if !ok || raw == "" || raw == "null" {
		return nil, nil
	}
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "[{") {
		var legacy []Oferta
		if err := json.Unmarshal([]byte(raw), &legacy); err != nil {
			return nil, err
		}
		out := make([]OfertaID, 0, len(legacy))
		for _, o := range legacy {
			if o.ID != "" {
				out = append(out, o.ID)
			}
		}
		return out, nil
	}
	var ids []OfertaID
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func (s *Catalog) loadOfertas(ctx context.Context, ids []OfertaID) ([]Oferta, error) {
	out := make([]Oferta, 0, len(ids))
	for _, id := range ids {
		o, ok, err := s.getOferta(ctx, id)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, o)
		}
	}
	return out, nil
}

func (s *Catalog) addOfertaToDocumento(ctx context.Context, documentoID DocumentoID, ofertaID OfertaID) error {
	ids, err := s.listAssocIDs(ctx, documentoID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id == ofertaID {
			return s.saveJSON(ctx, ofertasDocKey(documentoID), ids)
		}
	}
	ids = append(ids, ofertaID)
	return s.saveJSON(ctx, ofertasDocKey(documentoID), ids)
}

func (s *Catalog) removeOfertaFromDocumento(ctx context.Context, documentoID DocumentoID, ofertaID OfertaID) error {
	ids, err := s.listAssocIDs(ctx, documentoID)
	if err != nil {
		return err
	}
	next := ids[:0]
	for _, id := range ids {
		if id != ofertaID {
			next = append(next, id)
		}
	}
	return s.saveJSON(ctx, ofertasDocKey(documentoID), next)
}

func (s *Catalog) refreshProdutoIndexForDocumento(ctx context.Context, documentoID DocumentoID, produtoIDs ...ProdutoID) error {
	for _, pid := range produtoIDs {
		if pid == "" {
			continue
		}
		if err := s.c.SRem(ctx, ofertasProdutoKey(pid), string(documentoID)); err != nil {
			return err
		}
	}
	ofertas, err := s.ListOfertasByDocumento(ctx, documentoID)
	if err != nil {
		return err
	}
	present := map[ProdutoID]struct{}{}
	for _, o := range ofertas {
		if o.ProdutoID != "" {
			present[o.ProdutoID] = struct{}{}
		}
	}
	for pid := range present {
		if err := s.c.SAdd(ctx, ofertasProdutoKey(pid), string(documentoID)); err != nil {
			return err
		}
	}
	return nil
}

func produtoIDsFromOfertas(ofertas []Oferta) map[ProdutoID]struct{} {
	out := make(map[ProdutoID]struct{})
	for _, o := range ofertas {
		if o.ProdutoID != "" {
			out[o.ProdutoID] = struct{}{}
		}
	}
	return out
}

func (s *Catalog) getJSON(ctx context.Context, key string, out any) (bool, error) {
	raw, ok, err := s.c.Get(ctx, key)
	if err != nil || !ok {
		return ok, err
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Catalog) saveJSON(ctx context.Context, key string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.c.Set(ctx, key, string(b))
}

func (s *Catalog) idsFromSetOrKeys(ctx context.Context, setKey, pattern, prefix string) ([]string, error) {
	ids, err := s.c.SMembers(ctx, setKey)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		keys, err := s.c.Keys(ctx, pattern)
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			if strings.HasPrefix(key, prefix) {
				ids = append(ids, strings.TrimPrefix(key, prefix))
			}
		}
	}
	sort.Strings(ids)
	return ids, nil
}
