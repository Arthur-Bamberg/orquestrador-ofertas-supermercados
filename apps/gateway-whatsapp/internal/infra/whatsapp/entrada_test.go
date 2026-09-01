package whatsapp_test

import (
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/whatsapp"
)

func TestEntrada_ignoraVazioSemTipo(t *testing.T) {
	if _, ok := whatsapp.Entrada(whatsapp.Inbound{ChatJID: "x", SenderJID: "y"}); ok {
		t.Fatal("vazio")
	}
}

func TestEntrada_fromMeComTexto(t *testing.T) {
	got, ok := whatsapp.Entrada(whatsapp.Inbound{FromMe: true, Texto: "oi", ChatJID: "a", SenderJID: "a", ProvedorID: "1"})
	if !ok || !got.FromMe || got.Corpo != "oi" {
		t.Fatalf("%+v ok=%v", got, ok)
	}
}

func TestEntrada_reacaoSemTextoAindaEntra(t *testing.T) {
	got, ok := whatsapp.Entrada(whatsapp.Inbound{
		ProvedorID: "id1",
		ChatJID:    "5511999999999@s.whatsapp.net",
		SenderJID:  "5511999999999@s.whatsapp.net",
		Tipo:       domain.MensagemReacao,
		Payload:    `{"alvo":"x"}`,
	})
	if !ok || got.Tipo != domain.MensagemReacao {
		t.Fatalf("%+v ok=%v", got, ok)
	}
}

func TestEntrada_grupoComImagem(t *testing.T) {
	got, ok := whatsapp.Entrada(whatsapp.Inbound{
		ProvedorID: "id1",
		ChatJID:    "120363abc@g.us",
		SenderJID:  "5511999999999@s.whatsapp.net",
		Grupo:      true,
		Texto:      "encarte",
		Midia:      &domain.MidiaBytes{Tipo: domain.MidiaImagem, Conteudo: []byte{1}},
	})
	if !ok || !got.Grupo || got.Midia == nil || got.Corpo != "encarte" {
		t.Fatalf("%+v ok=%v", got, ok)
	}
}

func TestEntrada_propagaAssuntoDoGrupo(t *testing.T) {
	got, ok := whatsapp.Entrada(whatsapp.Inbound{
		ProvedorID:   "id1",
		ChatJID:      "120363abc@g.us",
		SenderJID:    "5511999999999@s.whatsapp.net",
		Grupo:        true,
		Texto:        "oi",
		ConversaNome: "Bruna e amigos",
	})
	if !ok || got.ConversaNome != "Bruna e amigos" || !got.Grupo {
		t.Fatalf("%+v ok=%v", got, ok)
	}
}

func TestEntrada_historicoMarcaOrigemEStatusBroadcast(t *testing.T) {
	got, ok := whatsapp.Entrada(whatsapp.Inbound{
		ProvedorID: "s1",
		ChatJID:    "status@broadcast",
		SenderJID:  "5511999999999@s.whatsapp.net",
		Texto:      "story",
		Origem:     domain.OrigemHistorico,
	})
	if !ok || !got.Status || got.Origem != domain.OrigemHistorico {
		t.Fatalf("%+v ok=%v", got, ok)
	}
}
