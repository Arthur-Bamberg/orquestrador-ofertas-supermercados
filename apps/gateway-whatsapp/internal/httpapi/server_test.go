package httpapi_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/httpapi"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/canal"
)

func TestHealthReadyEEnvios(t *testing.T) {
	gw, disconnected := newGW(t, true), newGW(t, false)
	h := httpapi.New(gw, "secret", "")

	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/health", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("health %d", res.Code)
	}

	res = httptest.NewRecorder()
	httpapi.New(disconnected, "secret", "").ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/ready", nil))
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

func TestRastroHTTP_conversasMensagensMidiaCORS(t *testing.T) {
	gw := newGW(t, true)
	ctx := context.Background()
	old := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	mid := time.Date(2026, 8, 1, 11, 0, 0, 0, time.UTC)
	novo := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	primeira, err := gw.Receber(ctx, application.Entrada{
		ProvedorID:   "wamid.1",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		Corpo:        "um",
		PushName:     "Ana",
		CriadoEm:     old,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gw.Receber(ctx, application.Entrada{
		ProvedorID:   "wamid.2",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		Corpo:        "dois",
		PushName:     "Ana",
		CriadoEm:     mid,
	}); err != nil {
		t.Fatal(err)
	}
	comMidia, err := gw.Receber(ctx, application.Entrada{
		ProvedorID:   "wamid.3",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		Corpo:        "foto",
		PushName:     "Ana",
		CriadoEm:     novo,
		Midia: &domain.MidiaBytes{
			Tipo:     domain.MidiaImagem,
			Filename: "a.jpg",
			MIME:     "image/jpeg",
			Conteudo: []byte{1, 2, 3},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gw.Receber(ctx, application.Entrada{
		ProvedorID:   "wamid.g",
		ConversaJID:  "120363abc@g.us",
		RemetenteJID: "5511888888888",
		Grupo:        true,
		Corpo:        "grupo",
		PushName:     "Beto",
		CriadoEm:     novo.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	h := httpapi.New(gw, "secret", "http://localhost:5173")
	req := httptest.NewRequest(http.MethodGet, "/conversas", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("list %d %s", res.Code, res.Body.String())
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("cors=%q", got)
	}
	var conversas []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&conversas); err != nil {
		t.Fatal(err)
	}
	if len(conversas) != 2 {
		t.Fatalf("conversas=%v", conversas)
	}
	if conversas[0]["tipo"] != "grupo" || conversas[0]["permitido"] != false {
		t.Fatalf("grupo primeiro %+v", conversas[0])
	}
	if conversas[1]["permitido"] != true {
		t.Fatalf("direta permitido %+v", conversas[1])
	}
	if _, ok := conversas[0]["jid"]; !ok {
		t.Fatal("jid camelCase")
	}

	res = httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/conversas?tipo=grupo", nil))
	conversas = nil
	if err := json.NewDecoder(res.Body).Decode(&conversas); err != nil {
		t.Fatal(err)
	}
	if len(conversas) != 1 || conversas[0]["tipo"] != "grupo" {
		t.Fatalf("filtro tipo %v", conversas)
	}

	cid := string(primeira.Conversa.ID)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/conversas/"+cid, nil))
	if res.Code != http.StatusOK {
		t.Fatalf("get conversa %d %s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/conversas/"+cid+"/mensagens?limit=2", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("msgs %d %s", res.Code, res.Body.String())
	}
	var msgs []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&msgs); err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 || msgs[0]["corpo"] != "dois" || msgs[1]["corpo"] != "foto" {
		t.Fatalf("pagina recente cronologica %v", msgs)
	}
	antes := msgs[0]["criadoEm"].(string)
	antesID := msgs[0]["id"].(string)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/conversas/"+cid+"/mensagens?limit=2&antesCriadoEm="+antes+"&antesId="+antesID, nil))
	msgs = nil
	if err := json.NewDecoder(res.Body).Decode(&msgs); err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0]["corpo"] != "um" {
		t.Fatalf("cursor anteriores %v", msgs)
	}

	res = httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/mensagens/"+string(comMidia.Mensagem.ID)+"/midia", nil))
	if res.Code != http.StatusOK || res.Header().Get("Content-Type") != "image/jpeg" || res.Body.String() != "\x01\x02\x03" {
		t.Fatalf("midia %d %q %q", res.Code, res.Header().Get("Content-Type"), res.Body.String())
	}

	opt := httptest.NewRequest(http.MethodOptions, "/conversas", nil)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, opt)
	if res.Code != http.StatusNoContent {
		t.Fatalf("options %d", res.Code)
	}
	if !strings.Contains(res.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Fatalf("allow headers %q", res.Header().Get("Access-Control-Allow-Headers"))
	}
}

func TestCanalHTTP_estadoQRDesparear(t *testing.T) {
	pngMagic := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	fake := &situacaoCanal{sit: domain.CanalSituacao{
		Estado: domain.CanalPendente,
		QR:     "2@example-pairing-code",
	}}
	gw := application.New(application.Deps{
		Allow:  domain.NovaAllowlist("5511999999999"),
		Repo:   newMemRepo(),
		Canal:  fake,
		Midias: memMidia{files: map[string][]byte{}},
		NewID:  func() string { return "h-1" },
	})
	h := httpapi.New(gw, "secret", "")

	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/canal", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("GET sem token %d", res.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/canal", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("GET pendente %d %s", res.Code, res.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["estado"] != string(domain.CanalPendente) || body["jid"] != "" {
		t.Fatalf("pendente %+v", body)
	}
	raw, err := base64.StdEncoding.DecodeString(body["qrPngBase64"])
	if err != nil || !bytes.HasPrefix(raw, pngMagic) {
		t.Fatalf("QR PNG %v %q", err, body["qrPngBase64"])
	}

	fake.sit = domain.CanalSituacao{Estado: domain.CanalConectado, JID: "5511988887777@s.whatsapp.net"}
	req = httptest.NewRequest(http.MethodGet, "/canal", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	body = map[string]string{}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["estado"] != string(domain.CanalConectado) || body["jid"] != "5511988887777@s.whatsapp.net" || body["qrPngBase64"] != "" {
		t.Fatalf("conectado %+v", body)
	}

	fake.sit = domain.CanalSituacao{Estado: domain.CanalDesconectado, JID: "5511988887777@s.whatsapp.net"}
	req = httptest.NewRequest(http.MethodGet, "/canal", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	body = map[string]string{}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["estado"] != string(domain.CanalDesconectado) || body["jid"] == "" || body["qrPngBase64"] != "" {
		t.Fatalf("desconectado %+v", body)
	}

	res = httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/canal/desparear", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("POST sem token %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/canal/desparear", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("desparear %d %s", res.Code, res.Body.String())
	}
	if fake.calls != 1 {
		t.Fatalf("Desparear calls=%d", fake.calls)
	}
	body = map[string]string{}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["estado"] != string(domain.CanalPendente) || body["qrPngBase64"] == "" {
		t.Fatalf("após Desparear %+v", body)
	}

	fake.desparear = domain.ErrSemPareamento
	req = httptest.NewRequest(http.MethodPost, "/canal/desparear", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("sem Pareamento %d", res.Code)
	}

	stubGW := newGW(t, true)
	h = httpapi.New(stubGW, "secret", "")
	req = httptest.NewRequest(http.MethodPost, "/canal/desparear", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("stub Desparear %d %s", res.Code, res.Body.String())
	}
}

type situacaoCanal struct {
	sit       domain.CanalSituacao
	desparear error
	calls     int
}

func (c *situacaoCanal) Enviar(context.Context, domain.JID, string, *domain.MidiaBytes) (string, error) {
	return "stub", nil
}

func (c *situacaoCanal) Conectado() bool { return c.sit.Estado == domain.CanalConectado }

func (c *situacaoCanal) Situacao() domain.CanalSituacao { return c.sit }

func (c *situacaoCanal) Desparear(context.Context) error {
	c.calls++
	if c.desparear != nil {
		return c.desparear
	}
	c.sit = domain.CanalSituacao{Estado: domain.CanalPendente, QR: "2@example-pairing-code"}
	return nil
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
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Mensagem
	for _, msg := range m.msgs {
		if msg.ConversaID == conversaID {
			out = append(out, msg)
		}
	}
	return out, nil
}

func (m *memRepo) ListarConversas(_ context.Context) ([]domain.ConversaResumo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.ConversaResumo, 0, len(m.conversas))
	for _, c := range m.conversas {
		r := domain.ConversaResumo{Conversa: c}
		for _, msg := range m.msgs {
			if msg.ConversaID != c.ID {
				continue
			}
			r.TotalMensagens++
			cp := msg
			if r.UltimaMensagem == nil || msg.CriadoEm.After(r.UltimaMensagem.CriadoEm) {
				r.UltimaMensagem = &cp
			}
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		ti, tj := time.Time{}, time.Time{}
		if out[i].UltimaMensagem != nil {
			ti = out[i].UltimaMensagem.CriadoEm
		}
		if out[j].UltimaMensagem != nil {
			tj = out[j].UltimaMensagem.CriadoEm
		}
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return string(out[i].ID) < string(out[j].ID)
	})
	return out, nil
}

func (m *memRepo) GetMensagem(_ context.Context, id domain.MensagemID) (domain.Mensagem, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.msgs[id]
	return msg, ok, nil
}

func (m *memRepo) GetConversa(_ context.Context, id domain.ConversaID) (domain.Conversa, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.conversas[id]
	return c, ok, nil
}

func (m *memRepo) ConversaPorJID(_ context.Context, jid domain.JID) (domain.Conversa, bool, error) {
	return domain.Conversa{}, false, nil
}

func (m *memRepo) MarcarRecibo(_ context.Context, provedorID string, status domain.StatusEnvio) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byProv[provedorID]
	if !ok {
		return false, nil
	}
	msg := m.msgs[id]
	msg.Status = status
	m.msgs[id] = msg
	return true, nil
}
