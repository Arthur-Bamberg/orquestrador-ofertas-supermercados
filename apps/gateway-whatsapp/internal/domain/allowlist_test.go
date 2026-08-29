package domain

import "testing"

func TestAllowlist_permiteJIDDaConversaERejeitaOutro(t *testing.T) {
	lista := NovaAllowlist("5511999999999, 120363abc@g.us")

	if !lista.PermiteConversa("5511999999999") {
		t.Fatal("E.164 da pessoa deveria ser aceito")
	}
	if !lista.PermiteConversa("5511999999999@s.whatsapp.net") {
		t.Fatal("JID da pessoa deveria ser aceito")
	}
	if !lista.PermiteConversa("120363ABC@g.us") {
		t.Fatal("JID do grupo deveria ser aceito (case-insensitive)")
	}
	if lista.PermiteConversa("5511888888888") {
		t.Fatal("pessoa fora da lista não deveria ser aceita")
	}
	if lista.PermiteConversa("120363outro@g.us") {
		t.Fatal("grupo fora da lista não deveria ser aceito")
	}
}
