package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

type Deps struct {
	Catalog  *store.Catalog
	Vitrines map[store.MercadoID]domain.Vitrine
	Hoje     func() string
	Token    string
}

type Server struct {
	deps Deps
}

func New(deps Deps) http.Handler {
	s := &Server{deps: deps}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /coletas", s.coletas)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) authorized(r *http.Request) bool {
	got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return s.deps.Token != "" && got == s.deps.Token
}

type coletaBody struct {
	Termo     string          `json:"termo"`
	MercadoID store.MercadoID `json:"mercadoId"`
}

func (s *Server) vitrinesDaColeta(mercadoID store.MercadoID) (map[store.MercadoID]domain.Vitrine, error) {
	if mercadoID == "" {
		return s.deps.Vitrines, nil
	}
	vitrine, ok := s.deps.Vitrines[mercadoID]
	if !ok {
		return nil, fmt.Errorf("mercado %s sem vitrine", mercadoID)
	}
	return map[store.MercadoID]domain.Vitrine{mercadoID: vitrine}, nil
}

func (s *Server) coletas(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "não autorizado"})
		return
	}
	var body coletaBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json inválido"})
		return
	}
	if strings.TrimSpace(body.Termo) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "termo obrigatório"})
		return
	}
	hoje := s.deps.Hoje
	if hoje == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dia da Coleta não configurado"})
		return
	}
	vitrines, err := s.vitrinesDaColeta(body.MercadoID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	ofertas, err := application.Coletar(r.Context(), s.deps.Catalog, vitrines, body.Termo, hoje())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if ofertas == nil {
		ofertas = []store.Oferta{}
	}
	writeJSON(w, http.StatusOK, map[string][]store.Oferta{"ofertas": ofertas})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
