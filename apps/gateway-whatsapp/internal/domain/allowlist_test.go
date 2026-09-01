package domain

import "testing"

func TestIdentidade_preferePNEguardaLID(t *testing.T) {
	jid, lid := Identidade("5115@lid", "555199784248", "5115@lid")
	if jid != NormalizarJID("555199784248") {
		t.Fatalf("jid=%s", jid)
	}
	if lid != "5115@lid" {
		t.Fatalf("lid=%s", lid)
	}
	soLid, soLid2 := Identidade("5115@lid", "", "")
	if soLid != "5115@lid" || soLid2 != "5115@lid" {
		t.Fatalf("%s %s", soLid, soLid2)
	}
}

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

func TestAllowlist_casaDeviceSuffixEVarianteBRDoNonoDigito(t *testing.T) {
	lista := NovaAllowlist("5551999784248")

	if !lista.PermiteConversa("555199784248@s.whatsapp.net") {
		t.Fatal("número sem o 9º dígito BR deveria casar com a lista")
	}
	if !lista.PermiteConversa("555199784248:61@s.whatsapp.net") {
		t.Fatal("JID com device deveria casar com a lista")
	}
	if !lista.PermiteConversa("5551999784248", "5115473883234@lid") {
		t.Fatal("qualquer candidato da Conversa deveria bastar")
	}
	if lista.PermiteConversa("5115473883234@lid") {
		t.Fatal("LID sozinho sem PN não deveria casar com E.164")
	}
}
