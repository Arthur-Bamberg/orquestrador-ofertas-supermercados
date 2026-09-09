package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-api/internal/config"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

type Server struct {
	catalog *store.Catalog
	cfg     config.Config
}

func New(catalog *store.Catalog, cfg config.Config) http.Handler {
	s := &Server{catalog: catalog, cfg: cfg}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/mercados", s.listMercados)
	mux.HandleFunc("POST /api/mercados", s.createMercado)
	mux.HandleFunc("GET /api/mercados/{id}", s.getMercado)
	mux.HandleFunc("PUT /api/mercados/{id}", s.updateMercado)
	mux.HandleFunc("DELETE /api/mercados/{id}", s.deleteMercado)

	mux.HandleFunc("GET /api/fontes", s.listFontes)
	mux.HandleFunc("POST /api/fontes", s.createFonte)
	mux.HandleFunc("GET /api/fontes/{id}", s.getFonte)
	mux.HandleFunc("PUT /api/fontes/{id}", s.updateFonte)
	mux.HandleFunc("DELETE /api/fontes/{id}", s.deleteFonte)

	mux.HandleFunc("GET /api/produtos", s.listProdutos)
	mux.HandleFunc("POST /api/produtos", s.createProduto)
	mux.HandleFunc("GET /api/produtos/{id}", s.getProduto)
	mux.HandleFunc("PUT /api/produtos/{id}", s.updateProduto)
	mux.HandleFunc("DELETE /api/produtos/{id}", s.deleteProduto)

	mux.HandleFunc("GET /api/marcas", s.listMarcas)
	mux.HandleFunc("POST /api/marcas", s.createMarca)
	mux.HandleFunc("GET /api/marcas/{id}", s.getMarca)
	mux.HandleFunc("PUT /api/marcas/{id}", s.updateMarca)
	mux.HandleFunc("DELETE /api/marcas/{id}", s.deleteMarca)

	mux.HandleFunc("GET /api/documentos", s.listDocumentos)
	mux.HandleFunc("POST /api/documentos", s.createDocumento)
	mux.HandleFunc("GET /api/documentos/{id}", s.getDocumento)
	mux.HandleFunc("PUT /api/documentos/{id}", s.updateDocumento)
	mux.HandleFunc("DELETE /api/documentos/{id}", s.deleteDocumento)
	mux.HandleFunc("GET /api/documentos/{id}/falhas", s.getFalhas)
	mux.HandleFunc("PUT /api/documentos/{id}/falhas", s.putFalhas)
	mux.HandleFunc("POST /api/documentos/{id}/falhas", s.putFalhas)
	mux.HandleFunc("DELETE /api/documentos/{id}/falhas", s.deleteFalhas)

	mux.HandleFunc("GET /api/ofertas", s.listOfertas)
	mux.HandleFunc("POST /api/ofertas", s.createOferta)
	mux.HandleFunc("GET /api/ofertas/{id}", s.getOferta)
	mux.HandleFunc("PUT /api/ofertas/{id}", s.updateOferta)
	mux.HandleFunc("DELETE /api/ofertas/{id}", s.deleteOferta)

	mux.HandleFunc("GET /api/usos-extrator", s.listUsos)
	mux.HandleFunc("POST /api/usos-extrator", s.createUso)
	mux.HandleFunc("GET /api/usos-extrator/{documentoId}/{tentativa}", s.getUso)
	mux.HandleFunc("PUT /api/usos-extrator/{documentoId}/{tentativa}", s.updateUso)
	mux.HandleFunc("DELETE /api/usos-extrator/{documentoId}/{tentativa}", s.deleteUso)

	mux.HandleFunc("GET /api/documentos/{id}/artefatos", s.listArtefatos)
	mux.HandleFunc("GET /api/documentos/{id}/artefatos/{tentativa}/{path...}", s.downloadArtefato)
	mux.HandleFunc("PUT /api/documentos/{id}/artefatos/{tentativa}/{path...}", s.uploadArtefato)
	mux.HandleFunc("DELETE /api/documentos/{id}/artefatos/{tentativa}", s.deleteTentativa)
	mux.HandleFunc("DELETE /api/documentos/{id}/artefatos/{tentativa}/{path...}", s.deleteArtefato)

	mux.HandleFunc("POST /api/ops/run", s.enqueueRun)
	mux.HandleFunc("POST /api/ops/discover", s.enqueueDiscover)
	mux.HandleFunc("POST /api/ops/reprocess", s.enqueueReprocess)
	mux.HandleFunc("POST /api/ops/testar", s.testarMercado)
	mux.HandleFunc("GET /api/ops", s.listOps)
	mux.HandleFunc("GET /api/ops/{id}", s.getOp)
	mux.HandleFunc("POST /api/ops/{id}/cancel", s.cancelOp)

	return s.withCORS(mux)
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.CORSOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", s.cfg.CORSOrigin)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) listMercados(w http.ResponseWriter, r *http.Request) {
	items, err := s.catalog.ListMercados(r.Context())
	writeResult(w, items, err)
}

func (s *Server) createMercado(w http.ResponseWriter, r *http.Request) {
	var m store.Mercado
	if !decode(w, r, &m) {
		return
	}
	if m.ID == "" {
		m.ID = store.MercadoID(newID())
	}
	if err := s.catalog.SaveMercado(r.Context(), m); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) getMercado(w http.ResponseWriter, r *http.Request) {
	item, ok, err := s.catalog.GetMercado(r.Context(), store.MercadoID(r.PathValue("id")))
	writeOptional(w, item, ok, err)
}

func (s *Server) updateMercado(w http.ResponseWriter, r *http.Request) {
	var m store.Mercado
	if !decode(w, r, &m) {
		return
	}
	m.ID = store.MercadoID(r.PathValue("id"))
	writeResult(w, m, s.catalog.SaveMercado(r.Context(), m))
}

func (s *Server) deleteMercado(w http.ResponseWriter, r *http.Request) {
	writeNoContent(w, s.catalog.DeleteMercado(r.Context(), store.MercadoID(r.PathValue("id"))))
}

func (s *Server) listFontes(w http.ResponseWriter, r *http.Request) {
	items, err := s.catalog.ListFontes(r.Context())
	if err == nil && r.URL.Query().Get("mercadoId") != "" {
		mercadoID := store.MercadoID(r.URL.Query().Get("mercadoId"))
		items = filter(items, func(f store.Fonte) bool { return f.MercadoID == mercadoID })
	}
	writeResult(w, items, err)
}

func (s *Server) createFonte(w http.ResponseWriter, r *http.Request) {
	var f store.Fonte
	if !decode(w, r, &f) {
		return
	}
	if f.ID == "" {
		f.ID = store.FonteID(newID())
	}
	if err := s.catalog.SaveFonte(r.Context(), f); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (s *Server) getFonte(w http.ResponseWriter, r *http.Request) {
	item, ok, err := s.catalog.GetFonte(r.Context(), store.FonteID(r.PathValue("id")))
	writeOptional(w, item, ok, err)
}

func (s *Server) updateFonte(w http.ResponseWriter, r *http.Request) {
	var f store.Fonte
	if !decode(w, r, &f) {
		return
	}
	f.ID = store.FonteID(r.PathValue("id"))
	writeResult(w, f, s.catalog.SaveFonte(r.Context(), f))
}

func (s *Server) deleteFonte(w http.ResponseWriter, r *http.Request) {
	writeNoContent(w, s.catalog.DeleteFonte(r.Context(), store.FonteID(r.PathValue("id"))))
}

func (s *Server) listProdutos(w http.ResponseWriter, r *http.Request) {
	items, err := s.catalog.ListProdutos(r.Context())
	q := r.URL.Query()
	if err == nil && q.Get("nome") != "" {
		nome := q.Get("nome")
		items = filter(items, func(p store.Produto) bool {
			return containsFold(p.Nome, nome) || containsFold(p.NomeNorm, nome)
		})
	}
	if err == nil && q.Get("categoria") != "" {
		cat := q.Get("categoria")
		items = filter(items, func(p store.Produto) bool {
			for _, c := range p.Categorias {
				if containsFold(c, cat) {
					return true
				}
			}
			return false
		})
	}
	writeResult(w, items, err)
}

func (s *Server) createProduto(w http.ResponseWriter, r *http.Request) {
	var p store.Produto
	if !decode(w, r, &p) {
		return
	}
	if p.ID == "" {
		p.ID = store.ProdutoID(newID())
	}
	if err := s.catalog.SaveProduto(r.Context(), p); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) getProduto(w http.ResponseWriter, r *http.Request) {
	item, ok, err := s.catalog.GetProduto(r.Context(), store.ProdutoID(r.PathValue("id")))
	writeOptional(w, item, ok, err)
}

func (s *Server) updateProduto(w http.ResponseWriter, r *http.Request) {
	var p store.Produto
	if !decode(w, r, &p) {
		return
	}
	p.ID = store.ProdutoID(r.PathValue("id"))
	writeResult(w, p, s.catalog.SaveProduto(r.Context(), p))
}

func (s *Server) deleteProduto(w http.ResponseWriter, r *http.Request) {
	writeNoContent(w, s.catalog.DeleteProduto(r.Context(), store.ProdutoID(r.PathValue("id"))))
}

func (s *Server) listMarcas(w http.ResponseWriter, r *http.Request) {
	items, err := s.catalog.ListMarcas(r.Context())
	writeResult(w, items, err)
}

func (s *Server) createMarca(w http.ResponseWriter, r *http.Request) {
	var m store.Marca
	if !decode(w, r, &m) {
		return
	}
	if m.ID == "" {
		m.ID = store.MarcaID(newID())
	}
	if err := s.catalog.SaveMarca(r.Context(), m); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) getMarca(w http.ResponseWriter, r *http.Request) {
	item, ok, err := s.catalog.GetMarca(r.Context(), store.MarcaID(r.PathValue("id")))
	writeOptional(w, item, ok, err)
}

func (s *Server) updateMarca(w http.ResponseWriter, r *http.Request) {
	var m store.Marca
	if !decode(w, r, &m) {
		return
	}
	m.ID = store.MarcaID(r.PathValue("id"))
	writeResult(w, m, s.catalog.SaveMarca(r.Context(), m))
}

func (s *Server) deleteMarca(w http.ResponseWriter, r *http.Request) {
	writeNoContent(w, s.catalog.DeleteMarca(r.Context(), store.MarcaID(r.PathValue("id"))))
}

func (s *Server) listDocumentos(w http.ResponseWriter, r *http.Request) {
	items, err := s.catalog.ListDocumentos(r.Context())
	q := r.URL.Query()
	if err == nil && q.Get("fonteId") != "" {
		fonteID := store.FonteID(q.Get("fonteId"))
		items = filter(items, func(d store.Documento) bool { return d.FonteID == fonteID })
	}
	if err == nil && q.Get("estado") != "" {
		estado := store.EstadoDocumento(q.Get("estado"))
		items = filter(items, func(d store.Documento) bool { return d.Estado == estado })
	}
	if err == nil && q.Get("dia") != "" {
		dia := q.Get("dia")
		items = filter(items, func(d store.Documento) bool { return d.Dia == dia })
	}
	writeResult(w, items, err)
}

func (s *Server) createDocumento(w http.ResponseWriter, r *http.Request) {
	var d store.Documento
	if !decode(w, r, &d) {
		return
	}
	if d.ID == "" {
		d.ID = store.DocumentoID(newID())
	}
	if d.Atualizado.IsZero() {
		d.Atualizado = time.Now().UTC()
	}
	if err := s.catalog.SaveDocumento(r.Context(), d); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (s *Server) getDocumento(w http.ResponseWriter, r *http.Request) {
	item, ok, err := s.catalog.GetDocumento(r.Context(), store.DocumentoID(r.PathValue("id")))
	writeOptional(w, item, ok, err)
}

func (s *Server) updateDocumento(w http.ResponseWriter, r *http.Request) {
	var d store.Documento
	if !decode(w, r, &d) {
		return
	}
	d.ID = store.DocumentoID(r.PathValue("id"))
	if d.Atualizado.IsZero() {
		d.Atualizado = time.Now().UTC()
	}
	writeResult(w, d, s.catalog.SaveDocumento(r.Context(), d))
}

func (s *Server) deleteDocumento(w http.ResponseWriter, r *http.Request) {
	writeNoContent(w, s.catalog.DeleteDocumento(r.Context(), store.DocumentoID(r.PathValue("id"))))
}

func (s *Server) getFalhas(w http.ResponseWriter, r *http.Request) {
	items, err := s.catalog.GetFalhasDocumento(r.Context(), store.DocumentoID(r.PathValue("id")))
	writeResult(w, items, err)
}

func (s *Server) putFalhas(w http.ResponseWriter, r *http.Request) {
	var items []store.FalhaExtracao
	if !decode(w, r, &items) {
		return
	}
	writeResult(w, items, s.catalog.SaveFalhasDocumento(r.Context(), store.DocumentoID(r.PathValue("id")), items))
}

func (s *Server) deleteFalhas(w http.ResponseWriter, r *http.Request) {
	writeNoContent(w, s.catalog.DeleteFalhasDocumento(r.Context(), store.DocumentoID(r.PathValue("id"))))
}

type ofertaPayload struct {
	store.Oferta
	DocumentoIDs []store.DocumentoID `json:"documentoIds"`
}

func (p *ofertaPayload) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &p.Oferta); err != nil {
		return err
	}
	var extra struct {
		DocumentoIDs []store.DocumentoID `json:"documentoIds"`
	}
	if err := json.Unmarshal(data, &extra); err != nil {
		return err
	}
	p.DocumentoIDs = extra.DocumentoIDs
	return nil
}

func (p ofertaPayload) documentoIDs() []store.DocumentoID {
	if len(p.DocumentoIDs) > 0 {
		return p.DocumentoIDs
	}
	if p.DocumentoID != "" {
		return []store.DocumentoID{p.DocumentoID}
	}
	return nil
}

func (s *Server) listOfertas(w http.ResponseWriter, r *http.Request) {
	items, err := s.catalog.ListOfertas(r.Context())
	q := r.URL.Query()
	if err == nil && q.Get("documentoId") != "" {
		items, err = s.catalog.ListOfertasByDocumento(r.Context(), store.DocumentoID(q.Get("documentoId")))
	}
	if err == nil && q.Get("produtoId") != "" {
		produtoID := store.ProdutoID(q.Get("produtoId"))
		items = filter(items, func(o store.Oferta) bool { return o.ProdutoID == produtoID })
	}
	if err == nil && q.Get("mercadoId") != "" {
		mercadoID := store.MercadoID(q.Get("mercadoId"))
		items = filter(items, func(o store.Oferta) bool { return o.MercadoID == mercadoID })
	}
	if err == nil && q.Get("marcaId") != "" {
		marcaID := store.MarcaID(q.Get("marcaId"))
		items = filter(items, func(o store.Oferta) bool { return o.MarcaID != nil && *o.MarcaID == marcaID })
	}
	if err == nil && q.Get("texto") != "" {
		items, err = s.filterOfertasTexto(r, items)
	}
	writeResult(w, items, err)
}

func (s *Server) filterOfertasTexto(r *http.Request, items []store.Oferta) ([]store.Oferta, error) {
	texto := r.URL.Query().Get("texto")
	produtos, err := s.catalog.ListProdutos(r.Context())
	if err != nil {
		return nil, err
	}
	marcas, err := s.catalog.ListMarcas(r.Context())
	if err != nil {
		return nil, err
	}
	produtoMatch := map[store.ProdutoID]bool{}
	for _, p := range produtos {
		produtoMatch[p.ID] = containsFold(p.Nome, texto) || containsFold(p.NomeNorm, texto)
	}
	marcaMatch := map[store.MarcaID]bool{}
	for _, m := range marcas {
		marcaMatch[m.ID] = containsFold(m.Nome, texto) || containsFold(m.NomeNorm, texto)
	}
	return filter(items, func(o store.Oferta) bool {
		if produtoMatch[o.ProdutoID] {
			return true
		}
		return o.MarcaID != nil && marcaMatch[*o.MarcaID]
	}), nil
}

func (s *Server) createOferta(w http.ResponseWriter, r *http.Request) {
	var p ofertaPayload
	if !decode(w, r, &p) {
		return
	}
	if p.ID == "" {
		p.ID = store.OfertaID(newID())
	}
	if err := s.catalog.SaveOferta(r.Context(), p.Oferta, p.documentoIDs()); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) getOferta(w http.ResponseWriter, r *http.Request) {
	item, ok, err := s.catalog.GetOferta(r.Context(), store.OfertaID(r.PathValue("id")))
	writeOptional(w, item, ok, err)
}

func (s *Server) updateOferta(w http.ResponseWriter, r *http.Request) {
	var p ofertaPayload
	if !decode(w, r, &p) {
		return
	}
	p.ID = store.OfertaID(r.PathValue("id"))
	writeResult(w, p, s.catalog.SaveOferta(r.Context(), p.Oferta, p.documentoIDs()))
}

func (s *Server) deleteOferta(w http.ResponseWriter, r *http.Request) {
	writeNoContent(w, s.catalog.DeleteOferta(r.Context(), store.OfertaID(r.PathValue("id"))))
}

func (s *Server) listUsos(w http.ResponseWriter, r *http.Request) {
	items, err := s.catalog.ListUsosExtrator(r.Context(), store.DocumentoID(r.URL.Query().Get("documentoId")))
	writeResult(w, items, err)
}

func (s *Server) createUso(w http.ResponseWriter, r *http.Request) {
	var uso store.UsoExtrator
	if !decode(w, r, &uso) {
		return
	}
	if err := s.catalog.SaveUsoExtrator(r.Context(), uso); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, uso)
}

func (s *Server) getUso(w http.ResponseWriter, r *http.Request) {
	item, ok, err := s.catalog.GetUsoExtrator(r.Context(), store.DocumentoID(r.PathValue("documentoId")), r.PathValue("tentativa"))
	writeOptional(w, item, ok, err)
}

func (s *Server) updateUso(w http.ResponseWriter, r *http.Request) {
	var uso store.UsoExtrator
	if !decode(w, r, &uso) {
		return
	}
	uso.DocumentoID = store.DocumentoID(r.PathValue("documentoId"))
	uso.Tentativa = r.PathValue("tentativa")
	writeResult(w, uso, s.catalog.SaveUsoExtrator(r.Context(), uso))
}

func (s *Server) deleteUso(w http.ResponseWriter, r *http.Request) {
	writeNoContent(w, s.catalog.DeleteUsoExtrator(r.Context(), store.DocumentoID(r.PathValue("documentoId")), r.PathValue("tentativa")))
}

type opRequest struct {
	FonteID     store.FonteID     `json:"fonteId"`
	DocumentoID store.DocumentoID `json:"documentoId"`
	MercadoID   store.MercadoID   `json:"mercadoId"`
	Termo       string            `json:"termo"`
}

func (s *Server) enqueueRun(w http.ResponseWriter, r *http.Request) {
	s.enqueueOp(w, r, store.OperacaoPipeline{ID: newID(), Kind: store.OperacaoRun})
}

func (s *Server) enqueueDiscover(w http.ResponseWriter, r *http.Request) {
	var req opRequest
	if !decode(w, r, &req) {
		return
	}
	s.enqueueOp(w, r, store.OperacaoPipeline{ID: newID(), Kind: store.OperacaoDiscover, FonteID: req.FonteID})
}

func (s *Server) enqueueReprocess(w http.ResponseWriter, r *http.Request) {
	var req opRequest
	if !decode(w, r, &req) {
		return
	}
	s.enqueueOp(w, r, store.OperacaoPipeline{ID: newID(), Kind: store.OperacaoReprocess, DocumentoID: req.DocumentoID})
}

type testarResponse struct {
	Kind      string                  `json:"kind"`
	MercadoID store.MercadoID         `json:"mercadoId"`
	FonteID   store.FonteID           `json:"fonteId"`
	Operacao  *store.OperacaoPipeline `json:"operacao,omitempty"`
	Ofertas   []store.Oferta          `json:"ofertas,omitempty"`
}

func (s *Server) testarMercado(w http.ResponseWriter, r *http.Request) {
	var req opRequest
	if !decode(w, r, &req) {
		return
	}
	if req.MercadoID == "" {
		writeError(w, fmt.Errorf("%w: mercadoId obrigatório", store.ErrInvalid))
		return
	}
	if _, ok, err := s.catalog.GetMercado(r.Context(), req.MercadoID); err != nil {
		writeError(w, err)
		return
	} else if !ok {
		writeError(w, fmt.Errorf("%w: mercado %s", store.ErrNotFound, req.MercadoID))
		return
	}
	fonte, err := s.fonteParaTeste(r.Context(), req.MercadoID)
	if err != nil {
		writeError(w, err)
		return
	}
	if fonte.TipoOuEncarte() == store.TipoSite {
		s.testarColeta(w, r, fonte, strings.TrimSpace(req.Termo))
		return
	}
	s.testarDiscover(w, r, fonte)
}

func (s *Server) fonteParaTeste(ctx context.Context, mercadoID store.MercadoID) (store.Fonte, error) {
	fontes, err := s.catalog.ListFontes(ctx)
	if err != nil {
		return store.Fonte{}, err
	}
	var doMercado, ativas []store.Fonte
	for _, f := range fontes {
		if f.MercadoID != mercadoID {
			continue
		}
		doMercado = append(doMercado, f)
		if f.IsAtiva() {
			ativas = append(ativas, f)
		}
	}
	pool := ativas
	if len(pool) == 0 {
		pool = doMercado
	}
	if len(pool) == 0 {
		return store.Fonte{}, fmt.Errorf("%w: mercado %s sem Fonte", store.ErrInvalid, mercadoID)
	}
	return pool[0], nil
}

func (s *Server) testarDiscover(w http.ResponseWriter, r *http.Request, fonte store.Fonte) {
	op := store.OperacaoPipeline{ID: newID(), Kind: store.OperacaoDiscover, FonteID: fonte.ID}
	if err := s.catalog.EnqueueOperacao(r.Context(), op); err != nil {
		writeError(w, err)
		return
	}
	got, _, _ := s.catalog.GetOperacao(r.Context(), op.ID)
	writeJSON(w, http.StatusAccepted, testarResponse{
		Kind: "discover", MercadoID: fonte.MercadoID, FonteID: fonte.ID, Operacao: &got,
	})
}

func (s *Server) testarColeta(w http.ResponseWriter, r *http.Request, fonte store.Fonte, termo string) {
	if termo == "" {
		writeError(w, fmt.Errorf("%w: termo obrigatório para Fonte tipo site", store.ErrInvalid))
		return
	}
	if s.cfg.ColetaURL == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "COLETA_URL não configurada"})
		return
	}
	ofertas, err := s.chamarColeta(r, fonte.MercadoID, termo)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, testarResponse{
		Kind: "coleta", MercadoID: fonte.MercadoID, FonteID: fonte.ID, Ofertas: ofertas,
	})
}

func (s *Server) chamarColeta(r *http.Request, mercadoID store.MercadoID, termo string) ([]store.Oferta, error) {
	payload, err := json.Marshal(map[string]string{"termo": termo, "mercadoId": string(mercadoID)})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, s.cfg.ColetaURL+"/coletas", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.cfg.ColetaToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.ColetaToken)
	}
	res, err := coletaHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = res.Status
		}
		if res.StatusCode == http.StatusBadRequest {
			return nil, fmt.Errorf("%w: %s", store.ErrInvalid, msg)
		}
		return nil, fmt.Errorf("coleta HTTP %d: %s", res.StatusCode, msg)
	}
	var out struct {
		Ofertas []store.Oferta `json:"ofertas"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out.Ofertas == nil {
		out.Ofertas = []store.Oferta{}
	}
	return out.Ofertas, nil
}

var coletaHTTPClient = &http.Client{Timeout: 2 * time.Minute}

func (s *Server) enqueueOp(w http.ResponseWriter, r *http.Request, op store.OperacaoPipeline) {
	if err := s.catalog.EnqueueOperacao(r.Context(), op); err != nil {
		writeError(w, err)
		return
	}
	got, _, _ := s.catalog.GetOperacao(r.Context(), op.ID)
	writeJSON(w, http.StatusAccepted, got)
}

func (s *Server) listOps(w http.ResponseWriter, r *http.Request) {
	items, err := s.catalog.ListOperacoes(r.Context())
	writeResult(w, items, err)
}

func (s *Server) getOp(w http.ResponseWriter, r *http.Request) {
	item, ok, err := s.catalog.GetOperacao(r.Context(), r.PathValue("id"))
	writeOptional(w, item, ok, err)
}

func (s *Server) cancelOp(w http.ResponseWriter, r *http.Request) {
	item, err := s.catalog.CancelOperacao(r.Context(), r.PathValue("id"), time.Now())
	writeResult(w, item, err)
}

type tentativaInfo struct {
	Tentativa string     `json:"tentativa"`
	Files     []fileInfo `json:"files"`
}

type fileInfo struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func (s *Server) listArtefatos(w http.ResponseWriter, r *http.Request) {
	doc, ok, err := s.catalog.GetDocumento(r.Context(), store.DocumentoID(r.PathValue("id")))
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, fmt.Errorf("%w: documento", store.ErrNotFound))
		return
	}
	root := s.documentoArtefatoDir(doc)
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		writeJSON(w, http.StatusOK, []tentativaInfo{})
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]tentativaInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info := tentativaInfo{Tentativa: entry.Name()}
		base := filepath.Join(root, entry.Name())
		err := filepath.WalkDir(base, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return walkErr
			}
			stat, err := d.Info()
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(base, path)
			if err != nil {
				return err
			}
			info.Files = append(info.Files, fileInfo{Path: filepath.ToSlash(rel), Size: stat.Size()})
			return nil
		})
		if err != nil {
			writeError(w, err)
			return
		}
		out = append(out, info)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) downloadArtefato(w http.ResponseWriter, r *http.Request) {
	path, ok := s.artefatoPath(w, r)
	if !ok {
		return
	}
	http.ServeFile(w, r, path)
}

func (s *Server) uploadArtefato(w http.ResponseWriter, r *http.Request) {
	path, ok := s.artefatoPath(w, r)
	if !ok {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		writeError(w, err)
		return
	}
	f, err := os.Create(path)
	if err != nil {
		writeError(w, err)
		return
	}
	defer f.Close()
	if _, err := io.Copy(f, r.Body); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": r.PathValue("path")})
}

func (s *Server) deleteTentativa(w http.ResponseWriter, r *http.Request) {
	doc, ok := s.documentoFromRequest(w, r)
	if !ok {
		return
	}
	path := filepath.Join(s.documentoArtefatoDir(doc), filepath.Clean(r.PathValue("tentativa")))
	writeNoContent(w, os.RemoveAll(path))
}

func (s *Server) deleteArtefato(w http.ResponseWriter, r *http.Request) {
	path, ok := s.artefatoPath(w, r)
	if !ok {
		return
	}
	writeNoContent(w, os.Remove(path))
}

func (s *Server) artefatoPath(w http.ResponseWriter, r *http.Request) (string, bool) {
	doc, ok := s.documentoFromRequest(w, r)
	if !ok {
		return "", false
	}
	rel := filepath.Clean(filepath.FromSlash(r.PathValue("path")))
	if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return "", false
	}
	tentativa := filepath.Clean(r.PathValue("tentativa"))
	if tentativa == "." || strings.HasPrefix(tentativa, "..") || filepath.IsAbs(tentativa) {
		http.Error(w, "invalid tentativa", http.StatusBadRequest)
		return "", false
	}
	return filepath.Join(s.documentoArtefatoDir(doc), tentativa, rel), true
}

func (s *Server) documentoFromRequest(w http.ResponseWriter, r *http.Request) (store.Documento, bool) {
	doc, ok, err := s.catalog.GetDocumento(r.Context(), store.DocumentoID(r.PathValue("id")))
	if err != nil {
		writeError(w, err)
		return store.Documento{}, false
	}
	if !ok {
		writeError(w, fmt.Errorf("%w: documento", store.ErrNotFound))
		return store.Documento{}, false
	}
	return doc, true
}

func (s *Server) documentoArtefatoDir(doc store.Documento) string {
	return filepath.Join(s.cfg.ArtefatoRoot, string(doc.FonteID), doc.Dia, doc.Filename)
}

func decode(w http.ResponseWriter, r *http.Request, out any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func writeOptional(w http.ResponseWriter, v any, ok bool, err error) {
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, store.ErrNotFound)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func writeResult(w http.ResponseWriter, v any, err error) {
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func writeNoContent(w http.ResponseWriter, err error) {
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, store.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, store.ErrDeleteBlocked), errors.Is(err, store.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, store.ErrInvalid):
		status = http.StatusBadRequest
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func filter[T any](in []T, keep func(T) bool) []T {
	out := in[:0]
	for _, item := range in {
		if keep(item) {
			out = append(out, item)
		}
	}
	return out
}

func containsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
