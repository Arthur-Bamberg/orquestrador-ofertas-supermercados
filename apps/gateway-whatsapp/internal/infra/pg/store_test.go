package pg_test

import (
	"context"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/pg/pgtest"
)

func TestMain(m *testing.M) {
	code := m.Run()
	pgtest.Stop()
	os.Exit(code)
}

func TestStore_receberGrupoEMidiaRoundtrip(t *testing.T) {
	repo := pgtest.New(t)
	var n atomic.Int64
	midias := &memMidia{files: map[string][]byte{}}
	gw := application.New(application.Deps{
		Allow:  domain.NovaAllowlist("120363abc@g.us"),
		Repo:   repo,
		Canal:  stubCanal{},
		Midias: midias,
		NewID: func() string {
			return t.Name() + "-" + itoa(n.Add(1))
		},
	})
	if _, err := repo.UpsertContato(context.Background(), domain.Contato{
		ID:         domain.ContatoID("aceite-grupo"),
		JID:        domain.NormalizarJID("5511999999999"),
		BoasVindas: true,
		Aceite:     true,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.pg1",
		ConversaJID:  "120363abc@g.us",
		RemetenteJID: "5511999999999",
		Grupo:        true,
		Corpo:        "caption",
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
	dup, err := gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.pg1",
		ConversaJID:  "120363abc@g.us",
		RemetenteJID: "5511999999999",
		Grupo:        true,
		Corpo:        "caption",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !dup.Duplicada || dup.Mensagem.ID != got.Mensagem.ID {
		t.Fatalf("dup %+v", dup)
	}
	msgs, err := gw.ListarMensagens(context.Background(), got.Conversa.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Corpo != "caption" || msgs[0].Midia == nil || msgs[0].Midia.Tipo != domain.MidiaImagem {
		t.Fatalf("msgs %+v", msgs)
	}
	if got.Conversa.Tipo != domain.ConversaGrupo {
		t.Fatalf("tipo=%s", got.Conversa.Tipo)
	}
}

func TestStore_receberConcorrenteMesmoProvedorNaoErro(t *testing.T) {
	repo := pgtest.New(t)
	var n atomic.Int64
	gw := application.New(application.Deps{
		Allow: domain.NovaAllowlist(""),
		Repo:  repo,
		Canal: stubCanal{},
		NewID: func() string {
			return t.Name() + "-" + itoa(n.Add(1))
		},
	})
	if _, err := repo.UpsertContato(context.Background(), domain.Contato{
		ID:         domain.ContatoID("aceite-race"),
		JID:        domain.NormalizarJID("5511999999999"),
		BoasVindas: true,
		Aceite:     true,
	}); err != nil {
		t.Fatal(err)
	}
	in := application.Entrada{
		ProvedorID:   "wamid.race",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		Corpo:        "oi",
	}
	const N = 32
	errs := make(chan error, N)
	for i := 0; i < N; i++ {
		go func() {
			_, err := gw.Receber(context.Background(), in)
			errs <- err
		}()
	}
	var failed error
	for i := 0; i < N; i++ {
		if err := <-errs; err != nil && failed == nil {
			failed = err
		}
	}
	if failed != nil {
		t.Fatalf("receber concorrente: %v", failed)
	}
	msgs, err := gw.ListarMensagens(context.Background(), mustConversa(t, gw, "5511999999999"))
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("mensagens=%d want 1", len(msgs))
	}
}

func mustConversa(t *testing.T, gw *application.Gateway, jid string) domain.ConversaID {
	t.Helper()
	items, err := gw.ListarConversas(context.Background(), application.FiltroConversas{})
	if err != nil {
		t.Fatal(err)
	}
	want := domain.NormalizarJID(jid)
	for _, c := range items {
		if c.JID == want {
			return c.ID
		}
	}
	t.Fatalf("conversa %s não encontrada", jid)
	return ""
}

func TestStore_listarConversasOrdenaPelaUltima(t *testing.T) {
	repo := pgtest.New(t)
	var n atomic.Int64
	gw := application.New(application.Deps{
		Allow: domain.NovaAllowlist(""),
		Repo:  repo,
		Canal: stubCanal{},
		NewID: func() string {
			return t.Name() + "-" + itoa(n.Add(1))
		},
	})
	ctx := context.Background()
	for i, jid := range []string{"5511999999999", "5511888888888"} {
		if _, err := repo.UpsertContato(ctx, domain.Contato{
			ID:         domain.ContatoID("aceite-list-" + itoa(int64(i+1))),
			JID:        domain.NormalizarJID(jid),
			BoasVindas: true,
			Aceite:     true,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := gw.Receber(ctx, application.Entrada{
		ProvedorID:   "wamid.old",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		Corpo:        "antiga",
		CriadoEm:     time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := gw.Receber(ctx, application.Entrada{
		ProvedorID:   "wamid.new",
		ConversaJID:  "5511888888888",
		RemetenteJID: "5511888888888",
		Corpo:        "nova",
		PushName:     "Beto",
		CriadoEm:     time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.ListarConversas(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].UltimaMensagem == nil || got[0].UltimaMensagem.Corpo != "nova" {
		t.Fatalf("ordem %+v", got)
	}
	if got[0].TotalMensagens != 1 {
		t.Fatalf("total=%d", got[0].TotalMensagens)
	}
}

func TestStore_boasVindasEAceitePersistem(t *testing.T) {
	repo := pgtest.New(t)
	var n atomic.Int64
	gw := application.New(application.Deps{
		Allow: domain.NovaAllowlist(""),
		Repo:  repo,
		Canal: stubCanal{},
		NewID: func() string {
			return t.Name() + "-" + itoa(n.Add(1))
		},
	})
	ctx := context.Background()
	first, err := gw.Receber(ctx, application.Entrada{
		ProvedorID:   "wamid.bv",
		ConversaJID:  "5511777777777",
		RemetenteJID: "5511777777777",
		Corpo:        "oi",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gw.Receber(ctx, application.Entrada{
		ProvedorID:   "wamid.um",
		ConversaJID:  "5511777777777",
		RemetenteJID: "5511777777777",
		Corpo:        "1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := gw.Receber(ctx, application.Entrada{
		ProvedorID:   "wamid.lista",
		ConversaJID:  "5511777777777",
		RemetenteJID: "5511777777777",
		Corpo:        "leite",
	}); err != nil {
		t.Fatal(err)
	}
	msgs, err := gw.ListarMensagens(ctx, first.Conversa.ID)
	if err != nil {
		t.Fatal(err)
	}
	temBoas, temConfirmacao := false, false
	for _, m := range msgs {
		if m.Direcao != domain.DirecaoSaida {
			continue
		}
		if strings.Contains(m.Corpo, "TERMOS DE USO") {
			temBoas = true
		}
		if m.Corpo == domain.TextoConfirmacaoAceite {
			temConfirmacao = true
		}
	}
	if !temBoas || !temConfirmacao {
		t.Fatalf("saidas incompletas (%d msgs)", len(msgs))
	}
}

type stubCanal struct{}

func (stubCanal) Enviar(context.Context, domain.Envio) (string, error) {
	return "stub", nil
}
func (stubCanal) Pronto() bool { return true }

func (stubCanal) Situacao() domain.CanalSituacao {
	return domain.CanalSituacao{Estado: domain.CanalPronto}
}

type memMidia struct{ files map[string][]byte }

func (m *memMidia) Guardar(_ context.Context, id domain.MensagemID, midia domain.MidiaBytes) (string, error) {
	path := "m/" + string(id)
	m.files[path] = append([]byte(nil), midia.Conteudo...)
	return path, nil
}

func (m *memMidia) Ler(_ context.Context, path string) ([]byte, error) { return m.files[path], nil }

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
