package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/operador"
)

type identificarBody struct {
	Nome  string `json:"nome"`
	Senha string `json:"senha"`
}

func (s *Server) identificar(w http.ResponseWriter, r *http.Request) {
	var body identificarBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json inválido"})
		return
	}
	id, err := s.ids.Identificar(r.Context(), body.Nome, body.Senha)
	if err != nil {
		if errors.Is(err, operador.ErrCredencial) || errors.Is(err, operador.ErrNome) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}
		writeError(w, err)
		return
	}
	operador.SetCookie(w, id.ID)
	writeJSON(w, http.StatusOK, map[string]string{"nome": id.Operador.Nome})
}

func (s *Server) sair(w http.ResponseWriter, r *http.Request) {
	if err := s.ids.Sair(r.Context(), operador.IDFromRequest(r)); err != nil {
		writeError(w, err)
		return
	}
	operador.ClearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) eu(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"nome": s.operadorNome(r)})
}

func (s *Server) requireOperador(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		switch r.URL.Path {
		case "/api/identificar", "/api/sair":
			next.ServeHTTP(w, r)
			return
		}
		if s.operadorNome(r) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "não identificado"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) operadorNome(r *http.Request) string {
	if s.ids == nil {
		return ""
	}
	op, ok, err := s.ids.OperadorPorIdentificacao(r.Context(), operador.IDFromRequest(r))
	if err != nil || !ok {
		return ""
	}
	return op.Nome
}
