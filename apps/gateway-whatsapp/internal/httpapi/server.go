package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/operador"
	"rsc.io/qr"
)

type Server struct {
	gw         *application.Gateway
	token      string
	corsOrigin string
	ids        operador.Consulta
}

func New(gw *application.Gateway, token, corsOrigin string, ids operador.Consulta) http.Handler {
	s := &Server{gw: gw, token: token, corsOrigin: corsOrigin, ids: ids}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /ready", s.ready)
	mux.HandleFunc("POST /envios", s.envios)
	mux.HandleFunc("GET /conversas", s.listarConversas)
	mux.HandleFunc("GET /conversas/{id}", s.obterConversa)
	mux.HandleFunc("GET /conversas/{id}/mensagens", s.listarMensagens)
	mux.HandleFunc("GET /mensagens/{id}/midia", s.obterMidia)
	mux.HandleFunc("GET /canal", s.canal)
	mux.HandleFunc("POST /canal/desparear", s.desparear)
	return s.withCORS(s.proteger(mux))
}

func (s *Server) proteger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health", "/ready", "/envios":
			next.ServeHTTP(w, r)
			return
		}
		if !s.requireOperador(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.corsOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", s.corsOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) ready(w http.ResponseWriter, _ *http.Request) {
	if s.gw == nil || !s.gw.Conectado() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "canal desconectado"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ready": true})
}

type envioBody struct {
	ConversaJID string `json:"conversaJid"`
	Corpo       string `json:"corpo"`
	Midia       *struct {
		Tipo           string `json:"tipo"`
		Filename       string `json:"filename"`
		MIME           string `json:"mime"`
		ConteudoBase64 string `json:"conteudoBase64"`
	} `json:"midia"`
}

func (s *Server) envios(w http.ResponseWriter, r *http.Request) {
	if !s.maquina(r) && !s.operador(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "não autorizado"})
		return
	}
	var body envioBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json inválido"})
		return
	}
	out := application.Saida{ConversaJID: body.ConversaJID, Corpo: body.Corpo}
	if body.Midia != nil && body.Midia.ConteudoBase64 != "" {
		raw, err := base64.StdEncoding.DecodeString(body.Midia.ConteudoBase64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mídia inválida"})
			return
		}
		out.Midia = &domain.MidiaBytes{
			Tipo:     domain.TipoMidia(body.Midia.Tipo),
			Filename: body.Midia.Filename,
			MIME:     body.Midia.MIME,
			Conteudo: raw,
		}
	}
	msg, err := s.gw.Enviar(r.Context(), out)
	if err != nil {
		if errors.Is(err, application.ErrNaoPermitido) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

func (s *Server) listarConversas(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := s.gw.ListarConversas(r.Context(), application.FiltroConversas{
		Tipo: domain.TipoConversa(q.Get("tipo")),
		Q:    q.Get("q"),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	out := make([]conversaJSON, 0, len(items))
	for _, item := range items {
		out = append(out, conversaToJSON(item))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) obterConversa(w http.ResponseWriter, r *http.Request) {
	item, ok, err := s.gw.ObterConversa(r.Context(), domain.ConversaID(r.PathValue("id")))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversa não encontrada"})
		return
	}
	writeJSON(w, http.StatusOK, conversaToJSON(item))
}

func (s *Server) listarMensagens(w http.ResponseWriter, r *http.Request) {
	id := domain.ConversaID(r.PathValue("id"))
	if _, ok, err := s.gw.ObterConversa(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	} else if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "conversa não encontrada"})
		return
	}
	q := r.URL.Query()
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limit inválido"})
			return
		}
		limit = n
	}
	var antes time.Time
	if raw := q.Get("antesCriadoEm"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "antesCriadoEm inválido"})
			return
		}
		antes = t
	}
	msgs, err := s.gw.ListarMensagensPagina(r.Context(), id, limit, antes, domain.MensagemID(q.Get("antesId")))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	out := make([]mensagemJSON, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, mensagemToJSON(m))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) canal(w http.ResponseWriter, r *http.Request) {
	if s.gw == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "canal desconectado"})
		return
	}
	writeCanal(w, http.StatusOK, s.gw.SituacaoCanal())
}

func (s *Server) desparear(w http.ResponseWriter, r *http.Request) {
	err := s.gw.Desparear(r.Context())
	if err != nil {
		if errors.Is(err, domain.ErrSemPareamento) || errors.Is(err, domain.ErrDesparearIndisponivel) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeCanal(w, http.StatusOK, s.gw.SituacaoCanal())
}

type canalJSON struct {
	Estado      string `json:"estado"`
	JID         string `json:"jid,omitempty"`
	QRPngBase64 string `json:"qrPngBase64,omitempty"`
}

func writeCanal(w http.ResponseWriter, status int, sit domain.CanalSituacao) {
	out := canalJSON{Estado: string(sit.Estado), JID: string(sit.JID)}
	if sit.QR != "" {
		png, err := pngQR(sit.QR)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "QR inválido"})
			return
		}
		out.QRPngBase64 = base64.StdEncoding.EncodeToString(png)
	}
	writeJSON(w, status, out)
}

func pngQR(code string) ([]byte, error) {
	c, err := qr.Encode(code, qr.M)
	if err != nil {
		return nil, err
	}
	return c.PNG(), nil
}

func (s *Server) obterMidia(w http.ResponseWriter, r *http.Request) {
	msg, ok, err := s.gw.ObterMensagem(r.Context(), domain.MensagemID(r.PathValue("id")))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !ok || msg.Midia == nil || msg.Midia.Path == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "mídia não encontrada"})
		return
	}
	raw, err := s.gw.LerMidia(r.Context(), msg.Midia.Path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "mídia não encontrada"})
		return
	}
	ctype := msg.Midia.MIME
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ctype)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (s *Server) requireOperador(w http.ResponseWriter, r *http.Request) bool {
	if s.operador(r) {
		return true
	}
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "não identificado"})
	return false
}

func (s *Server) operador(r *http.Request) bool {
	if s.ids == nil {
		return false
	}
	_, ok, err := s.ids.OperadorPorIdentificacao(r.Context(), operador.IDFromRequest(r))
	return err == nil && ok
}

func (s *Server) maquina(r *http.Request) bool {
	if s.token == "" {
		return false
	}
	h := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(h, "Bearer ")
	return ok && token == s.token
}

type conversaJSON struct {
	ID             string        `json:"id"`
	JID            string        `json:"jid"`
	JIDLID         string        `json:"jidLid,omitempty"`
	Tipo           string        `json:"tipo"`
	TotalMensagens int           `json:"totalMensagens"`
	Permitido      bool          `json:"permitido"`
	UltimaMensagem *mensagemJSON `json:"ultimaMensagem,omitempty"`
}

type mensagemJSON struct {
	ID         string     `json:"id"`
	ConversaID string     `json:"conversaId"`
	ContatoID  string     `json:"contatoId,omitempty"`
	Direcao    string     `json:"direcao"`
	Corpo      string     `json:"corpo"`
	ProvedorID string     `json:"provedorId,omitempty"`
	Status     string     `json:"status,omitempty"`
	CriadoEm   time.Time  `json:"criadoEm"`
	Origem     string     `json:"origem,omitempty"`
	PushName   string     `json:"pushName,omitempty"`
	Tipo       string     `json:"tipo,omitempty"`
	Payload    string     `json:"payload,omitempty"`
	Midia      *midiaJSON `json:"midia,omitempty"`
}

type midiaJSON struct {
	Tipo     string `json:"tipo"`
	Filename string `json:"filename,omitempty"`
	MIME     string `json:"mime,omitempty"`
}

func conversaToJSON(c application.ConversaResumo) conversaJSON {
	out := conversaJSON{
		ID:             string(c.ID),
		JID:            string(c.JID),
		JIDLID:         string(c.JIDLID),
		Tipo:           string(c.Tipo),
		TotalMensagens: c.TotalMensagens,
		Permitido:      c.Permitido,
	}
	if c.UltimaMensagem != nil {
		m := mensagemToJSON(*c.UltimaMensagem)
		out.UltimaMensagem = &m
	}
	return out
}

func mensagemToJSON(m domain.Mensagem) mensagemJSON {
	out := mensagemJSON{
		ID:         string(m.ID),
		ConversaID: string(m.ConversaID),
		ContatoID:  string(m.ContatoID),
		Direcao:    string(m.Direcao),
		Corpo:      m.Corpo,
		ProvedorID: m.ProvedorID,
		Status:     string(m.Status),
		CriadoEm:   m.CriadoEm,
		Origem:     string(m.Origem),
		PushName:   m.PushName,
		Tipo:       string(m.Tipo),
		Payload:    m.Payload,
	}
	if m.Midia != nil {
		out.Midia = &midiaJSON{Tipo: string(m.Midia.Tipo), Filename: m.Midia.Filename, MIME: m.Midia.MIME}
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
