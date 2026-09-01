package application_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
)

func TestListarConversas_ordenaPelaUltimaEMarcaPermitido(t *testing.T) {
	repo := newMemRepo()
	gw := application.New(application.Deps{
		Allow: domain.NovaAllowlist("5511999999999"),
		Repo:  repo,
		Canal: &stubCanal{},
		NewID: seqIDs(),
	})
	ctx := context.Background()
	old := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	if _, err := gw.Receber(ctx, application.Entrada{
		ProvedorID:   "wamid.allow",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		Corpo:        "oi allow",
		PushName:     "Ana",
		CriadoEm:     old,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := gw.Receber(ctx, application.Entrada{
		ProvedorID:   "wamid.out",
		ConversaJID:  "5511888888888",
		RemetenteJID: "5511888888888",
		Corpo:        "oi fora",
		PushName:     "Beto",
		CriadoEm:     newer,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := gw.ListarConversas(ctx, application.FiltroConversas{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d %+v", len(got), got)
	}
	if !strings.Contains(string(got[0].JID), "8888") {
		t.Fatalf("primeira deveria ser a mais recente: %+v", got[0])
	}
	if got[0].Permitido {
		t.Fatal("conversa fora da allowlist não é permitido enviar")
	}
	if !got[1].Permitido {
		t.Fatal("conversa da allowlist é permitido enviar")
	}
	if got[0].UltimaMensagem == nil || got[0].UltimaMensagem.Corpo != "oi fora" || got[0].UltimaMensagem.PushName != "Beto" {
		t.Fatalf("ultima %+v", got[0].UltimaMensagem)
	}
	if got[0].TotalMensagens != 1 || got[1].TotalMensagens != 1 {
		t.Fatalf("totais %d %d", got[0].TotalMensagens, got[1].TotalMensagens)
	}
}

func TestListarConversas_filtraTipoEQ(t *testing.T) {
	repo := newMemRepo()
	gw := application.New(application.Deps{
		Allow: domain.NovaAllowlist("5511999999999"),
		Repo:  repo,
		Canal: &stubCanal{},
		NewID: seqIDs(),
	})
	ctx := context.Background()
	if _, err := gw.Receber(ctx, application.Entrada{
		ProvedorID:   "wamid.d",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		Corpo:        "direta",
		PushName:     "Ana",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := gw.Receber(ctx, application.Entrada{
		ProvedorID:   "wamid.g",
		ConversaJID:  "120363abc@g.us",
		RemetenteJID: "5511888888888",
		Grupo:        true,
		Corpo:        "grupo",
		PushName:     "Beto",
	}); err != nil {
		t.Fatal(err)
	}

	grupos, err := gw.ListarConversas(ctx, application.FiltroConversas{Tipo: domain.ConversaGrupo})
	if err != nil {
		t.Fatal(err)
	}
	if len(grupos) != 1 || grupos[0].Tipo != domain.ConversaGrupo {
		t.Fatalf("tipo %+v", grupos)
	}
	porNome, err := gw.ListarConversas(ctx, application.FiltroConversas{Q: "BETO"})
	if err != nil {
		t.Fatal(err)
	}
	if len(porNome) != 1 || porNome[0].UltimaMensagem == nil || porNome[0].UltimaMensagem.PushName != "Beto" {
		t.Fatalf("q %+v", porNome)
	}
}
