package whatsapp

import "testing"

func TestAssuntosGrupo_lembraAssuntoDoGrupo(t *testing.T) {
	var a assuntosGrupo
	a.lembrar(" 120363abc@g.us ", " Bruna e amigos ")
	if got := a.nome("120363abc@g.us"); got != "Bruna e amigos" {
		t.Fatalf("nome=%q", got)
	}
	if a.nome("120363outro@g.us") != "" {
		t.Fatal("grupo desconhecido")
	}
	a.lembrar("", "Bruna")
	a.lembrar("x@g.us", "")
	if a.nome("x@g.us") != "" {
		t.Fatal("não deveria lembrar vazio")
	}
}
