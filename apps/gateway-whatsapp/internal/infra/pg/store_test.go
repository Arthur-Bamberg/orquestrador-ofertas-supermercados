package pg_test

import (
	"context"
	"os"
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

type stubCanal struct{}

func (stubCanal) Enviar(context.Context, domain.JID, string, *domain.MidiaBytes) (string, error) {
	return "stub", nil
}
func (stubCanal) Conectado() bool { return true }

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
