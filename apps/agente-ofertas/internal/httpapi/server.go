package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/domain"
)

type Server struct {
	ag    *application.Agente
	token string
}

func New(ag *application.Agente, token string) http.Handler {
	s := &Server{ag: ag, token: token}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /listas", s.listas)
	mux.HandleFunc("POST /interpretar", s.interpretar)
	mux.HandleFunc("POST /mcp", s.mcp)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) authorized(r *http.Request) bool {
	got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return s.token != "" && got == s.token
}

type listaBody struct {
	ConversaJID string `json:"conversaJid"`
	Corpo       string `json:"corpo"`
}

func (s *Server) listas(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "não autorizado"})
		return
	}
	var body listaBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json inválido"})
		return
	}
	if err := s.ag.Atender(r.Context(), body.ConversaJID, body.Corpo); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"ok": true})
}

type interpretarBody struct {
	Texto string `json:"texto"`
}

func (s *Server) interpretar(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "não autorizado"})
		return
	}
	var body interpretarBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json inválido"})
		return
	}
	texto, err := application.Interpretar(r.Context(), domain.ParseLista(body.Texto), s.ag.Catalogo(), s.ag.Agora())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"resposta": texto})
}

type mcpReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func (s *Server) mcp(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body"})
		return
	}
	var req mcpReq
	if err := json.Unmarshal(raw, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json inválido"})
		return
	}
	switch req.Method {
	case "initialize":
		writeJSON(w, http.StatusOK, map[string]any{
			"jsonrpc": "2.0", "id": req.ID,
			"result": map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]string{"name": "agente-ofertas", "version": "1"},
			},
		})
	case "tools/list":
		writeJSON(w, http.StatusOK, map[string]any{
			"jsonrpc": "2.0", "id": req.ID,
			"result": map[string]any{
				"tools": []map[string]any{{
					"name":        "interpretar_lista",
					"description": "Interpreta uma Lista contra Ofertas vigentes e devolve a Resposta.",
					"inputSchema": map[string]any{
						"type":       "object",
						"properties": map[string]any{"texto": map[string]string{"type": "string"}},
						"required":   []string{"texto"},
					},
				}},
			},
		})
	case "tools/call":
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		_ = json.Unmarshal(req.Params, &p)
		texto, _ := p.Arguments["texto"].(string)
		corpo, err := application.Interpretar(r.Context(), domain.ParseLista(texto), s.ag.Catalogo(), s.ag.Agora())
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0", "id": req.ID,
				"error": map[string]any{"code": -32000, "message": err.Error()},
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"jsonrpc": "2.0", "id": req.ID,
			"result": map[string]any{
				"content": []map[string]string{{"type": "text", "text": corpo}},
			},
		})
	default:
		writeJSON(w, http.StatusOK, map[string]any{
			"jsonrpc": "2.0", "id": req.ID,
			"error": map[string]any{"code": -32601, "message": "método desconhecido"},
		})
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
