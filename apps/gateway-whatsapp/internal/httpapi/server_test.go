package httpapi_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/operador"
)

const cookieOperador = "id-arthur"

func idsTeste() *operador.Mem {
	m := operador.NewMem()
	m.Fixar(cookieOperador, operador.Operador{ID: "op1", Nome: "arthur"})
	return m
}

func handlerGW(gw *application.Gateway, cors string) http.Handler {
	return httpapi.New(gw, "secret", cors, idsTeste())
}

func reqOperador(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(&http.Cookie{Name: operador.CookieName, Value: cookieOperador})
	return req
}

func TestHealthReadyEEnvios(t *testing.T) {
	gw, disconnected := newGW(t, true), newGW(t, false)
	h := httpapi.New(gw, "secret", "", nil)

	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/health", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("health %d", res.Code)
	}

	res = httptest.NewRecorder()
	httpapi.New(disconnected, "secret", "", nil).ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready off %d", res.Code)
	}
	res = httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("ready on %d", res.Code)
	}

	if _, err := gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.janela.http",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		Corpo:        "oi",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.janela.fora",
		ConversaJID:  "5511888888888",
		RemetenteJID: "5511888888888",
		Corpo:        "oi",
	}); err != nil {
		t.Fatal(err)
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

	hOp := handlerGW(gw, "")
	req = httptest.NewRequest(http.MethodPost, "/envios", bytes.NewBufferString(`{"conversaJid":"5511999999999","corpo":"ola"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: operador.CookieName, Value: cookieOperador})
	res = httptest.NewRecorder()
	hOp.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("envio Operador %d %s", res.Code, res.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/envios", bytes.NewBufferString(`{"conversaJid":"5511888888888","corpo":"fora"}`))
	req.AddCookie(&http.Cookie{Name: operador.CookieName, Value: cookieOperador})
	res = httptest.NewRecorder()
	hOp.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("Operador fora da Allowlist %d %s", res.Code, res.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/envios", bytes.NewBufferString(`{"conversaJid":"5511888888888","corpo":"resposta"}`))
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	hOp.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("Agente fora da Allowlist %d %s", res.Code, res.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/envios", bytes.NewBufferString(`{"conversaJid":"5511777777777","corpo":"tarde"}`))
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	hOp.ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("fora da Janela %d %s", res.Code, res.Body.String())
	}
}

func TestRastroHTTP_conversasMensagensMidiaCORS(t *testing.T) {
	gw := newGW(t, true, "5511999999999", "5511888888888")
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

	h := handlerGW(gw, "http://localhost:5173")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/conversas", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("rastro anônimo %d", res.Code)
	}
	req := httptest.NewRequest(http.MethodGet, "/conversas", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("rastro com token de máquina %d", res.Code)
	}
	req = reqOperador(http.MethodGet, "/conversas")
	res = httptest.NewRecorder()
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
	h.ServeHTTP(res, reqOperador(http.MethodGet, "/conversas?tipo=grupo"))
	conversas = nil
	if err := json.NewDecoder(res.Body).Decode(&conversas); err != nil {
		t.Fatal(err)
	}
	if len(conversas) != 1 || conversas[0]["tipo"] != "grupo" {
		t.Fatalf("filtro tipo %v", conversas)
	}

	cid := string(primeira.Conversa.ID)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, reqOperador(http.MethodGet, "/conversas/"+cid))
	if res.Code != http.StatusOK {
		t.Fatalf("get conversa %d %s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	h.ServeHTTP(res, reqOperador(http.MethodGet, "/conversas/"+cid+"/mensagens?limit=2"))
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
	h.ServeHTTP(res, reqOperador(http.MethodGet, "/conversas/"+cid+"/mensagens?limit=2&antesCriadoEm="+antes+"&antesId="+antesID))
	msgs = nil
	if err := json.NewDecoder(res.Body).Decode(&msgs); err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0]["corpo"] != "um" {
		t.Fatalf("cursor anteriores %v", msgs)
	}

	res = httptest.NewRecorder()
	h.ServeHTTP(res, reqOperador(http.MethodGet, "/mensagens/"+string(comMidia.Mensagem.ID)+"/midia"))
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

func TestCanalHTTP_estadoPronto(t *testing.T) {
	fake := &situacaoCanal{sit: domain.CanalSituacao{Estado: domain.CanalNaoConfigurado}}
	gw := application.New(application.Deps{
		Allow:  domain.NovaAllowlist("5511999999999"),
		Repo:   newMemRepo(),
		Canal:  fake,
		Midias: memMidia{files: map[string][]byte{}},
		NewID:  func() string { return "h-1" },
	})
	h := handlerGW(gw, "")

	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/canal", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("GET anônimo %d", res.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/canal", nil)
	req.Header.Set("Authorization", "Bearer secret")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("GET com token de máquina %d", res.Code)
	}

	req = reqOperador(http.MethodGet, "/canal")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("GET não configurado %d %s", res.Code, res.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["estado"] != string(domain.CanalNaoConfigurado) || body["jid"] != "" || body["qrPngBase64"] != "" {
		t.Fatalf("não configurado %+v", body)
	}

	fake.sit = domain.CanalSituacao{Estado: domain.CanalPronto, JID: "5511988887777@s.whatsapp.net"}
	req = reqOperador(http.MethodGet, "/canal")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	body = map[string]string{}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["estado"] != string(domain.CanalPronto) || body["jid"] != "5511988887777@s.whatsapp.net" {
		t.Fatalf("pronto %+v", body)
	}

	res = httptest.NewRecorder()
	h.ServeHTTP(res, reqOperador(http.MethodPost, "/canal/desparear"))
	if res.Code != http.StatusNotFound {
		t.Fatalf("desparear %d %s", res.Code, res.Body.String())
	}
}

func TestWebhookHTTP_verifyEReceber(t *testing.T) {
	gw := newGW(t, true, "5511999999999")
	h := httpapi.NewWith(gw, "secret", "", idsTeste(), httpapi.Options{VerifyToken: "verify-me", AppSecret: "app-secret"})

	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=errado&hub.challenge=abc", nil))
	if res.Code != http.StatusForbidden {
		t.Fatalf("verify errado %d", res.Code)
	}
	res = httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=verify-me&hub.challenge=abc", nil))
	if res.Code != http.StatusOK || res.Body.String() != "abc" {
		t.Fatalf("verify %d %q", res.Code, res.Body.String())
	}

	payload := `{"object":"whatsapp_business_account","entry":[{"changes":[{"value":{"contacts":[{"profile":{"name":"Ana"},"wa_id":"5511999999999"}],"messages":[{"from":"5511999999999","id":"wamid.hook","timestamp":"1757800000","type":"text","text":{"body":"tomate"}}]}}]}]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("sem assinatura %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	req.Header.Set("X-Hub-Signature-256", "sha256=dead")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("assinatura má %d", res.Code)
	}

	mac := hmac.New(sha256.New, []byte("app-secret"))
	mac.Write([]byte(payload))
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	req = httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	req.Header.Set("X-Hub-Signature-256", sig)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("webhook %d %s", res.Code, res.Body.String())
	}
	items, err := gw.ListarConversas(context.Background(), application.FiltroConversas{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].UltimaMensagem == nil || items[0].UltimaMensagem.Corpo != "tomate" {
		t.Fatalf("rastro %+v", items)
	}
}

type situacaoCanal struct {
	sit domain.CanalSituacao
}

func (c *situacaoCanal) Enviar(context.Context, domain.Envio) (string, error) {
	return "stub", nil
}

func (c *situacaoCanal) Pronto() bool { return c.sit.Estado == domain.CanalPronto }

func (c *situacaoCanal) Situacao() domain.CanalSituacao { return c.sit }

type readyCanal struct {
	canal.Stub
	on bool
}

func (c readyCanal) Pronto() bool { return c.on }

func newGW(t *testing.T, conectado bool, aceites ...string) *application.Gateway {
	t.Helper()
	n := 0
	var mu sync.Mutex
	repo := newMemRepo()
	for i, raw := range aceites {
		j := domain.NormalizarJID(raw)
		if _, err := repo.UpsertContato(context.Background(), domain.Contato{
			ID:         domain.ContatoID("aceite-" + strconv.Itoa(i) + "-" + string(j)),
			JID:        j,
			BoasVindas: true,
			Aceite:     true,
		}); err != nil {
			t.Fatal(err)
		}
	}
	return application.New(application.Deps{
		Allow:  domain.NovaAllowlist("5511999999999"),
		Repo:   repo,
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
		existing := m.contatos[id]
		if c.BoasVindas {
			existing.BoasVindas = true
		}
		if c.Aceite {
			existing.Aceite = true
		}
		m.contatos[id] = existing
		return existing, nil
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
