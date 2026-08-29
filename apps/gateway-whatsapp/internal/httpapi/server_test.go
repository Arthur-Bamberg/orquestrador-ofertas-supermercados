package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/httpapi"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/canal"
)

func TestHealthReadyEEnvios(t *testing.T) {
	gw, disconnected := newGW(t, true), newGW(t, false)
	h := httpapi.New(gw, "secret")

	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/health", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("health %d", res.Code)
	}

	res = httptest.NewRecorder()
	httpapi.New(disconnected, "secret").ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready off %d", res.Code)
	}
	res = httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("ready on %d", res.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/envios", bytes.NewBufferString(`{"conversaJid":"5511999999999","corpo":"oi"}`))
	req.Header.Set("Content-Type", "application/json")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("sem token %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/envios", bytes.NewBufferString(`{"conversaJid":"5511999999999","corpo":"oi","midia":{"tipo":"imagem","filename":"a.jpg","mime":"image/jpeg","conteudoBase64":"AQID"}}`))
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("envio %d %s", res.Code, res.Body.String())
	}
	var msg domain.Mensagem
	if err := json.NewDecoder(res.Body).Decode(&msg); err != nil {
		t.Fatal(err)
	}
	if msg.Corpo != "oi" || msg.Midia == nil || msg.Status != domain.StatusEnviado {
		t.Fatalf("%+v", msg)
	}
}

type readyCanal struct {
	canal.Stub
	on bool
}

func (c readyCanal) Conectado() bool { return c.on }

func newGW(t *testing.T, conectado bool) *application.Gateway {
	t.Helper()
	n := 0
	var mu sync.Mutex
	return application.New(application.Deps{
		Allow:  domain.NovaAllowlist("5511999999999"),
		Repo:   newMemRepo(),
		Canal:  &readyCanal{on: conectado},
		Midias: memMidia{files: map[string][]byte{}},
		NewID: func() string {
			mu.Lock()
			defer mu.Unlock()
			n++
			return "h-" + strconv.Itoa(n)
		},
	})
}

type memMidia struct{ files map[string][]byte }

func (m memMidia) Guardar(_ context.Context, id domain.MensagemID, midia domain.MidiaBytes) (string, error) {
	path := string(id)
	m.files[path] = append([]byte(nil), midia.Conteudo...)
	return path, nil
}

func (m memMidia) Ler(_ context.Context, path string) ([]byte, error) { return m.files[path], nil }

type memRepo struct {
	mu         sync.Mutex
	contatos   map[domain.ContatoID]domain.Contato
	contatoJID map[domain.JID]domain.ContatoID
	conversas  map[domain.ConversaID]domain.Conversa
	convJID    map[domain.JID]domain.ConversaID
	msgs       map[domain.MensagemID]domain.Mensagem
	byProv     map[string]domain.MensagemID
}

func newMemRepo() *memRepo {
	return &memRepo{
		contatos:   map[domain.ContatoID]domain.Contato{},
		contatoJID: map[domain.JID]domain.ContatoID{},
		conversas:  map[domain.ConversaID]domain.Conversa{},
		convJID:    map[domain.JID]domain.ConversaID{},
		msgs:       map[domain.MensagemID]domain.Mensagem{},
		byProv:     map[string]domain.MensagemID{},
	}
}

func (m *memRepo) UpsertContato(_ context.Context, c domain.Contato) (domain.Contato, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id, ok := m.contatoJID[c.JID]; ok {
		return m.contatos[id], nil
	}
	m.contatos[c.ID] = c
	m.contatoJID[c.JID] = c.ID
	return c, nil
}

func (m *memRepo) UpsertConversa(_ context.Context, c domain.Conversa) (domain.Conversa, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id, ok := m.convJID[c.JID]; ok {
		return m.conversas[id], nil
	}
	m.conversas[c.ID] = c
	m.convJID[c.JID] = c.ID
	return c, nil
}

func (m *memRepo) SalvarMensagem(_ context.Context, msg domain.Mensagem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs[msg.ID] = msg
	if msg.ProvedorID != "" {
		m.byProv[msg.ProvedorID] = msg.ID
	}
	return nil
}

func (m *memRepo) MensagemPorProvedor(_ context.Context, provedorID string) (domain.Mensagem, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byProv[provedorID]
	if !ok {
		return domain.Mensagem{}, false, nil
	}
	return m.msgs[id], true, nil
}

func (m *memRepo) ListarMensagens(_ context.Context, conversaID domain.ConversaID) ([]domain.Mensagem, error) {
	return nil, nil
}

func (m *memRepo) GetMensagem(_ context.Context, id domain.MensagemID) (domain.Mensagem, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.msgs[id]
	return msg, ok, nil
}

func (m *memRepo) GetConversa(_ context.Context, id domain.ConversaID) (domain.Conversa, bool, error) {
	return domain.Conversa{}, false, nil
}

func (m *memRepo) ConversaPorJID(_ context.Context, jid domain.JID) (domain.Conversa, bool, error) {
	return domain.Conversa{}, false, nil
}
