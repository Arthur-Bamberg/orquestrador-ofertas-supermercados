package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
)

type Server struct {
	gw    *application.Gateway
	token string
}

func New(gw *application.Gateway, token string) http.Handler {
	s := &Server{gw: gw, token: token}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /ready", s.ready)
	mux.HandleFunc("POST /envios", s.envios)
	return mux
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
	if !s.authorized(r) {
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

func (s *Server) authorized(r *http.Request) bool {
	if s.token == "" {
		return false
	}
	h := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(h, "Bearer ")
	return ok && token == s.token
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
