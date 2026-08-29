package application_test

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
)

func TestReceber_silencioForaDaAllowlist(t *testing.T) {
	fx := newGW(t, "5511999999999")
	got, err := fx.gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.x",
		ConversaJID:  "5511888888888",
		RemetenteJID: "5511888888888",
		Corpo:        "oi",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Aceita {
		t.Fatal("não deveria aceitar")
	}
	if len(fx.repo.msgs) != 0 {
		t.Fatalf("persistiu %d mensagens", len(fx.repo.msgs))
	}
}

func TestReceber_persisteTextoDeConversaNaAllowlist(t *testing.T) {
	fx := newGW(t, "5511999999999")
	got, err := fx.gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.1",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		Corpo:        "oi",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Aceita {
		t.Fatal("deveria aceitar")
	}
	msgs, err := fx.gw.ListarMensagens(context.Background(), got.Conversa.ID)
	if err != nil {
		t.Fatal(err)
	}
	entrada := primeira(msgs, domain.DirecaoEntrada)
	if entrada.Corpo != "oi" {
		t.Fatalf("entrada: %+v", msgs)
	}
	if fx.repo.conversas[got.Conversa.ID].Tipo != domain.ConversaDireta {
		t.Fatalf("tipo=%s", fx.repo.conversas[got.Conversa.ID].Tipo)
	}
}

func TestReceber_naoDuplicaMesmoProvedorID(t *testing.T) {
	fx := newGW(t, "5511999999999")
	in := application.Entrada{
		ProvedorID:   "wamid.1",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		Corpo:        "oi",
	}
	first, err := fx.gw.Receber(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	second, err := fx.gw.Receber(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Duplicada || second.Mensagem.ID != first.Mensagem.ID {
		t.Fatalf("duplicada=%v id=%s want %s", second.Duplicada, second.Mensagem.ID, first.Mensagem.ID)
	}
	if n := contarDirecao(fx.repo, domain.DirecaoEntrada); n != 1 {
		t.Fatalf("entradas=%d", n)
	}
}

func TestReceber_grupoAllowlistedViraConversaGrupo(t *testing.T) {
	fx := newGW(t, "120363abc@g.us")
	got, err := fx.gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.g1",
		ConversaJID:  "120363abc@g.us",
		RemetenteJID: "5511999999999",
		Grupo:        true,
		Corpo:        "lista: arroz",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Aceita || got.Conversa.Tipo != domain.ConversaGrupo {
		t.Fatalf("got aceita=%v tipo=%s", got.Aceita, got.Conversa.Tipo)
	}
	if fx.repo.contatos[got.Mensagem.ContatoID].JID != domain.NormalizarJID("5511999999999") {
		t.Fatalf("remetente %+v", fx.repo.contatos[got.Mensagem.ContatoID])
	}
}

func TestReceber_silencioEmGrupoForaDaAllowlistMesmoComRemetentePermitido(t *testing.T) {
	fx := newGW(t, "5511999999999")
	got, err := fx.gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.g2",
		ConversaJID:  "120363outro@g.us",
		RemetenteJID: "5511999999999",
		Grupo:        true,
		Corpo:        "oi",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Aceita || len(fx.repo.msgs) != 0 {
		t.Fatalf("aceita=%v msgs=%d", got.Aceita, len(fx.repo.msgs))
	}
}

func TestReceber_persisteMidiaComCaption(t *testing.T) {
	fx := newGW(t, "5511999999999")
	got, err := fx.gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.img",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		Corpo:        "encarte",
		Midia: &domain.MidiaBytes{
			Tipo:     domain.MidiaImagem,
			Filename: "pagina.jpg",
			MIME:     "image/jpeg",
			Conteudo: []byte{0xff, 0xd8, 0xff},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Mensagem.Midia == nil || got.Mensagem.Midia.Tipo != domain.MidiaImagem || got.Mensagem.Midia.Path == "" {
		t.Fatalf("midia %+v", got.Mensagem.Midia)
	}
	if got.Mensagem.Corpo != "encarte" {
		t.Fatalf("corpo=%s", got.Mensagem.Corpo)
	}
	b, err := fx.gw.LerMidia(context.Background(), got.Mensagem.Midia.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, []byte{0xff, 0xd8, 0xff}) {
		t.Fatalf("bytes=%v", b)
	}
}

func TestReceber_enviaAckNaMesmaConversa(t *testing.T) {
	fx := newGW(t, "120363abc@g.us")
	got, err := fx.gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.gack",
		ConversaJID:  "120363abc@g.us",
		RemetenteJID: "5511999999999",
		Grupo:        true,
		Corpo:        "oi grupo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fx.canal.envios) != 1 || fx.canal.envios[0].Corpo != "ack-teste" {
		t.Fatalf("envios=%+v", fx.canal.envios)
	}
	if fx.canal.envios[0].Destino != domain.NormalizarJID("120363abc@g.us") {
		t.Fatalf("destino=%s", fx.canal.envios[0].Destino)
	}
	saidas := 0
	for _, m := range fx.repo.msgs {
		if m.Direcao == domain.DirecaoSaida && m.Status == domain.StatusEnviado && m.Corpo == "ack-teste" && m.ConversaID == got.Conversa.ID {
			saidas++
		}
	}
	if saidas != 1 {
		t.Fatalf("acks persistidos=%d", saidas)
	}
}

func TestReceber_naoReenviaAckQuandoDuplicada(t *testing.T) {
	fx := newGW(t, "5511999999999")
	in := application.Entrada{
		ProvedorID:   "wamid.dupack",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		Corpo:        "oi",
	}
	if _, err := fx.gw.Receber(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.gw.Receber(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	if len(fx.canal.envios) != 1 {
		t.Fatalf("envios=%d", len(fx.canal.envios))
	}
}

func TestEnviar_textoEMidiaParaConversaAllowlisted(t *testing.T) {
	fx := newGW(t, "5511999999999")
	msg, err := fx.gw.Enviar(context.Background(), application.Saida{
		ConversaJID: "5511999999999",
		Corpo:       "resposta",
		Midia: &domain.MidiaBytes{
			Tipo:     domain.MidiaDocumento,
			Filename: "lista.txt",
			MIME:     "text/plain",
			Conteudo: []byte("arroz"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg.Direcao != domain.DirecaoSaida || msg.Status != domain.StatusEnviado || msg.Corpo != "resposta" {
		t.Fatalf("%+v", msg)
	}
	if msg.Midia == nil || msg.Midia.Tipo != domain.MidiaDocumento {
		t.Fatalf("midia %+v", msg.Midia)
	}
	if len(fx.canal.envios) != 1 || fx.canal.envios[0].Corpo != "resposta" {
		t.Fatalf("canal %+v", fx.canal.envios)
	}
}

func TestEnviar_marcaFalhouSeCanalErra(t *testing.T) {
	fx := newGW(t, "5511999999999")
	fx.canal.setErr(errors.New("whatsapp down"))
	msg, err := fx.gw.Enviar(context.Background(), application.Saida{
		ConversaJID: "5511999999999",
		Corpo:       "oi",
	})
	if err == nil {
		t.Fatal("esperava erro do canal")
	}
	if msg.Status != domain.StatusFalhou {
		t.Fatalf("status=%s", msg.Status)
	}
}

func TestEnviar_silencioForaDaAllowlist(t *testing.T) {
	fx := newGW(t, "5511999999999")
	_, err := fx.gw.Enviar(context.Background(), application.Saida{
		ConversaJID: "5511888888888",
		Corpo:       "spam",
	})
	if !errors.Is(err, application.ErrNaoPermitido) {
		t.Fatalf("err=%v", err)
	}
	if len(fx.canal.envios) != 0 {
		t.Fatalf("enviou %+v", fx.canal.envios)
	}
}

type fixture struct {
	gw    *application.Gateway
	repo  *memRepo
	canal *stubCanal
}

func newGW(t *testing.T, allowCSV string) fixture {
	t.Helper()
	repo := newMemRepo()
	canal := &stubCanal{}
	midias := &memMidia{files: map[string][]byte{}}
	gw := application.New(application.Deps{
		Allow:    domain.NovaAllowlist(allowCSV),
		Repo:     repo,
		Canal:    canal,
		Midias:   midias,
		AckTexto: "ack-teste",
		NewID:    seqIDs(),
	})
	return fixture{gw: gw, repo: repo, canal: canal}
}

func primeira(msgs []domain.Mensagem, d domain.Direcao) domain.Mensagem {
	for _, m := range msgs {
		if m.Direcao == d {
			return m
		}
	}
	return domain.Mensagem{}
}

func contarDirecao(repo *memRepo, d domain.Direcao) int {
	n := 0
	for _, m := range repo.msgs {
		if m.Direcao == d {
			n++
		}
	}
	return n
}

func seqIDs() func() string {
	var mu sync.Mutex
	n := 0
	return func() string {
		mu.Lock()
		defer mu.Unlock()
		n++
		return "id-" + strconv.Itoa(n)
	}
}

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
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.convJID[jid]
	if !ok {
		return domain.Conversa{}, false, nil
	}
	return m.conversas[id], true, nil
}

type stubCanal struct {
	mu        sync.Mutex
	envios    []stubEnvio
	err       error
	conectado bool
}

type stubEnvio struct {
	Destino domain.JID
	Corpo   string
	Midia   *domain.MidiaBytes
}

func (s *stubCanal) Enviar(_ context.Context, destino domain.JID, corpo string, midia *domain.MidiaBytes) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	var copyMidia *domain.MidiaBytes
	if midia != nil {
		c := *midia
		c.Conteudo = bytes.Clone(midia.Conteudo)
		copyMidia = &c
	}
	s.envios = append(s.envios, stubEnvio{Destino: destino, Corpo: corpo, Midia: copyMidia})
	return nil
}

func (s *stubCanal) Conectado() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conectado
}

func (s *stubCanal) setErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
}

type memMidia struct {
	mu    sync.Mutex
	files map[string][]byte
}

func (m *memMidia) Guardar(_ context.Context, mensagemID domain.MensagemID, midia domain.MidiaBytes) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	path := "midia/" + string(mensagemID)
	m.files[path] = bytes.Clone(midia.Conteudo)
	return path, nil
}

func (m *memMidia) Ler(_ context.Context, path string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.files[path]
	if !ok {
		return nil, errors.New("midia ausente")
	}
	return bytes.Clone(b), nil
}
