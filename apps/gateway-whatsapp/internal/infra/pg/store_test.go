package pg_test

import (
	"context"
	"os"
	"sync/atomic"
	"testing"

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

type stubCanal struct{}

func (stubCanal) Enviar(context.Context, domain.JID, string, *domain.MidiaBytes) error { return nil }
func (stubCanal) Conectado() bool                                                      { return true }

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
