package whatsapp_test

import (
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/whatsapp"
)

func TestEntrada_ignoraFromMeEVazio(t *testing.T) {
	if _, ok := whatsapp.Entrada(whatsapp.Inbound{FromMe: true, Texto: "oi"}); ok {
		t.Fatal("from me")
	}
	if _, ok := whatsapp.Entrada(whatsapp.Inbound{ChatJID: "x", SenderJID: "y"}); ok {
		t.Fatal("vazio")
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
