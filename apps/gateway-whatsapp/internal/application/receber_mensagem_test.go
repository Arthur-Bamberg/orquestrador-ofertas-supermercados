package application_test

import (
	"bytes"
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
)

func TestReceber_reusaConversaQuandoLIDDepoisPN(t *testing.T) {
	fx := newGW(t, "555199784248")
	first, err := fx.gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.a",
		ConversaJID:  "555199784248",
		ConversaLID:  "5115@lid",
		RemetenteJID: "555199784248",
		RemetenteLID: "5115@lid",
		Corpo:        "um",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := fx.gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.b",
		ConversaJID:  "5115@lid",
		RemetenteJID: "5115@lid",
		Corpo:        "dois",
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Conversa.ID != first.Conversa.ID {
		t.Fatalf("conversas %s vs %s", second.Conversa.ID, first.Conversa.ID)
	}
}

func TestReceber_preservaTextoSeGuardarMidiaFalhar(t *testing.T) {
	fx := newGW(t, "5511999999999")
	fx.midias.fail = true
	got, err := fx.gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.imgfail",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		Corpo:        "caption",
		Midia: &domain.MidiaBytes{
			Tipo:     domain.MidiaImagem,
			Filename: "a.jpg",
			MIME:     "image/jpeg",
			Conteudo: []byte{1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Mensagem.Corpo != "caption" {
		t.Fatalf("corpo=%s", got.Mensagem.Corpo)
	}
	if got.Mensagem.Midia == nil || got.Mensagem.Midia.Tipo != domain.MidiaImagem {
		t.Fatalf("midia %+v", got.Mensagem.Midia)
	}
}

func TestReceber_persisteForaDaAllowlistSemAck(t *testing.T) {
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
	if !got.Aceita {
		t.Fatal("deveria persistir")
	}
	if len(fx.canal.envios) != 0 {
		t.Fatalf("ack fora da allowlist: %+v", fx.canal.envios)
	}
	msgs, err := fx.gw.ListarMensagens(context.Background(), got.Conversa.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Corpo != "oi" {
		t.Fatalf("msgs %+v", msgs)
	}
}

func TestReceber_historicoNaAllowlistNaoEnviaAckEUsaTimestampDoProvedor(t *testing.T) {
	fx := newGW(t, "5511999999999")
	quando := time.Date(2024, 6, 1, 15, 4, 5, 0, time.UTC)
	got, err := fx.gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.hist",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		Corpo:        "arquivo",
		Origem:       domain.OrigemHistorico,
		CriadoEm:     quando,
		PushName:     "Ana",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fx.canal.envios) != 0 {
		t.Fatalf("ack no histórico: %+v", fx.canal.envios)
	}
	if !got.Mensagem.CriadoEm.Equal(quando) || got.Mensagem.Origem != domain.OrigemHistorico || got.Mensagem.PushName != "Ana" {
		t.Fatalf("%+v", got.Mensagem)
	}
}

func TestReceber_statusViraConversaStatusSemAck(t *testing.T) {
	fx := newGW(t, "5511999999999")
	got, err := fx.gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.st",
		ConversaJID:  "status@broadcast",
		RemetenteJID: "5511888888888",
		Status:       true,
		Corpo:        "story",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Conversa.Tipo != domain.ConversaStatus {
		t.Fatalf("tipo=%s", got.Conversa.Tipo)
	}
	if len(fx.canal.envios) != 0 {
		t.Fatalf("ack em status: %+v", fx.canal.envios)
	}
}

func TestReceber_fromMePersisteComoSaidaSemCanalNemAck(t *testing.T) {
	fx := newGW(t, "5511999999999")
	got, err := fx.gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.me",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		FromMe:       true,
		Corpo:        "eu disse",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Mensagem.Direcao != domain.DirecaoSaida || got.Mensagem.Status != domain.StatusEnviado {
		t.Fatalf("%+v", got.Mensagem)
	}
	if len(fx.canal.envios) != 0 {
		t.Fatalf("não deveria reenviar nem ack: %+v", fx.canal.envios)
	}
}

func TestReceber_fromMeDiretoNaoDisparaAgente(t *testing.T) {
	repo := newMemRepo()
	canal := &stubCanal{}
	ag := &stubAgente{}
	gw := application.New(application.Deps{
		Allow:    domain.NovaAllowlist("5511999999999"),
		Repo:     repo,
		Canal:    canal,
		Midias:   &memMidia{files: map[string][]byte{}},
		AckTexto: "ack-teste",
		Agente:   ag,
		NewID:    seqIDs(),
	})
	_, err := gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.me.direto",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		FromMe:       true,
		Grupo:        false,
		Corpo:        "tomate",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ag.calls) != 0 || len(canal.envios) != 0 {
		t.Fatalf("fromMe direto não deve acionar agente nem canal: ag=%+v canal=%+v", ag.calls, canal.envios)
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

func TestReceber_grupoForaDaAllowlistPersisteSemAckMesmoComRemetentePermitido(t *testing.T) {
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
	if !got.Aceita {
		t.Fatal("deveria persistir o grupo")
	}
	if len(fx.canal.envios) != 0 {
		t.Fatalf("ack no grupo fora da lista: %+v", fx.canal.envios)
	}
	if got.Conversa.Tipo != domain.ConversaGrupo {
		t.Fatalf("tipo=%s", got.Conversa.Tipo)
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

func TestReceber_notificaAgenteNoLugarDoAck(t *testing.T) {
	repo := newMemRepo()
	canal := &stubCanal{}
	midias := &memMidia{files: map[string][]byte{}}
	ag := &stubAgente{}
	gw := application.New(application.Deps{
		Allow:    domain.NovaAllowlist("5511999999999"),
		Repo:     repo,
		Canal:    canal,
		Midias:   midias,
		AckTexto: "ack-teste",
		Agente:   ag,
		NewID:    seqIDs(),
	})
	got, err := gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.agente",
		ConversaJID:  "5511999999999",
		RemetenteJID: "5511999999999",
		Corpo:        "leite e arroz",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(canal.envios) != 0 {
		t.Fatalf("ack com agente: %+v", canal.envios)
	}
	if len(ag.calls) != 1 || ag.calls[0].jid != string(got.Conversa.JID) || ag.calls[0].corpo != "leite e arroz" {
		t.Fatalf("%+v conversa=%s", ag.calls, got.Conversa.JID)
	}
}

func TestReceber_comAgenteNuncaEnviaAckMesmoComCorpoVazio(t *testing.T) {
	repo := newMemRepo()
	canal := &stubCanal{}
	ag := &stubAgente{}
	gw := application.New(application.Deps{
		Allow:    domain.NovaAllowlist("120363abc@g.us"),
		Repo:     repo,
		Canal:    canal,
		Midias:   &memMidia{files: map[string][]byte{}},
		AckTexto: "ack-teste",
		Agente:   ag,
		NewID:    seqIDs(),
	})
	_, err := gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.vazio",
		ConversaJID:  "120363abc@g.us",
		RemetenteJID: "5511999999999",
		Grupo:        true,
		FromMe:       true,
		Corpo:        "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(canal.envios) != 0 {
		t.Fatalf("com agente configurado nunca deve enviar ack: %+v", canal.envios)
	}
	if len(ag.calls) != 0 {
		t.Fatalf("não deve chamar agente para corpo vazio: %+v", ag.calls)
	}
}

func TestReceber_duplicadaAtualizaCorpoVazioEDisparaAgente(t *testing.T) {
	repo := newMemRepo()
	canal := &stubCanal{}
	ag := &stubAgente{}
	gw := application.New(application.Deps{
		Allow:    domain.NovaAllowlist("120363abc@g.us"),
		Repo:     repo,
		Canal:    canal,
		Midias:   &memMidia{files: map[string][]byte{}},
		AckTexto: "ack-teste",
		Agente:   ag,
		NewID:    seqIDs(),
	})
	_, err := gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.duplicada.retry",
		ConversaJID:  "120363abc@g.us",
		RemetenteJID: "5511999999999",
		Grupo:        true,
		FromMe:       true,
		Corpo:        "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ag.calls) != 0 {
		t.Fatalf("não deveria chamar agente para corpo vazio: %+v", ag.calls)
	}

	got2, err := gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.duplicada.retry",
		ConversaJID:  "120363abc@g.us",
		RemetenteJID: "5511999999999",
		Grupo:        true,
		FromMe:       true,
		Corpo:        "cenoura e abobrinha",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got2.Duplicada {
		t.Fatal("esperava mensagem marcada como duplicada")
	}
	if got2.Mensagem.Corpo != "cenoura e abobrinha" {
		t.Fatalf("esperava corpo atualizado, obteve %q", got2.Mensagem.Corpo)
	}
	if len(ag.calls) != 1 || ag.calls[0].corpo != "cenoura e abobrinha" {
		t.Fatalf("esperava agente disparado na atualização do corpo: %+v", ag.calls)
	}
}



func TestReceber_foraDaAllowlistNaoChamaAgente(t *testing.T) {
	repo := newMemRepo()
	canal := &stubCanal{}
	ag := &stubAgente{}
	gw := application.New(application.Deps{
		Allow:    domain.NovaAllowlist("5511999999999"),
		Repo:     repo,
		Canal:    canal,
		Midias:   &memMidia{files: map[string][]byte{}},
		AckTexto: "ack-teste",
		Agente:   ag,
		NewID:    seqIDs(),
	})
	if _, err := gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.fora",
		ConversaJID:  "5511888888888",
		RemetenteJID: "5511888888888",
		Corpo:        "leite",
	}); err != nil {
		t.Fatal(err)
	}
	if len(ag.calls) != 0 || len(canal.envios) != 0 {
		t.Fatalf("agente=%+v canal=%+v", ag.calls, canal.envios)
	}
}

func TestReceber_fromMeEmGrupoNaAllowlistDisparaAgente(t *testing.T) {
	repo := newMemRepo()
	canal := &stubCanal{}
	ag := &stubAgente{}
	gw := application.New(application.Deps{
		Allow:    domain.NovaAllowlist("120363abc@g.us"),
		Repo:     repo,
		Canal:    canal,
		Midias:   &memMidia{files: map[string][]byte{}},
		AckTexto: "ack-teste",
		Agente:   ag,
		NewID:    seqIDs(),
	})
	got, err := gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.fromme.grupo",
		ConversaJID:  "120363abc@g.us",
		RemetenteJID: "5511999999999",
		Grupo:        true,
		FromMe:       true,
		Corpo:        "tomate e banana",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ag.calls) != 1 || ag.calls[0].jid != string(got.Conversa.JID) || ag.calls[0].corpo != "tomate e banana" {
		t.Fatalf("agente não disparou para fromMe em grupo permitido: %+v", ag.calls)
	}
}

func TestReceber_fromMeEmGrupoSemAgenteEnviaAck(t *testing.T) {
	fx := newGW(t, "120363abc@g.us")
	got, err := fx.gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.fromme.ack",
		ConversaJID:  "120363abc@g.us",
		RemetenteJID: "5511999999999",
		Grupo:        true,
		FromMe:       true,
		Corpo:        "oi grupo meu texto",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fx.canal.envios) != 1 || fx.canal.envios[0].Corpo != "ack-teste" {
		t.Fatalf("esperava ack para fromMe em grupo permitido sem agente: %+v", fx.canal.envios)
	}
	if fx.canal.envios[0].Destino != domain.NormalizarJID(string(got.Conversa.JID)) {
		t.Fatalf("destino incorreto: %s", fx.canal.envios[0].Destino)
	}
}

func TestReceber_fromMeEmGrupoForaDaAllowlistNaoDispara(t *testing.T) {
	repo := newMemRepo()
	canal := &stubCanal{}
	ag := &stubAgente{}
	gw := application.New(application.Deps{
		Allow:    domain.NovaAllowlist("120363outro@g.us"),
		Repo:     repo,
		Canal:    canal,
		Midias:   &memMidia{files: map[string][]byte{}},
		AckTexto: "ack-teste",
		Agente:   ag,
		NewID:    seqIDs(),
	})
	_, err := gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   "wamid.fromme.fora",
		ConversaJID:  "120363abc@g.us",
		RemetenteJID: "5511999999999",
		Grupo:        true,
		FromMe:       true,
		Corpo:        "tomate e arroz",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ag.calls) != 0 || len(canal.envios) != 0 {
		t.Fatalf("não deveria disparar fora da allowlist: ag=%+v canal=%+v", ag.calls, canal.envios)
	}
}

func TestReceber_mensagemEnviadaPeloGatewayNaoGeraLoop(t *testing.T) {
	repo := newMemRepo()
	canal := &stubCanal{}
	ag := &stubAgente{}
	gw := application.New(application.Deps{
		Allow:    domain.NovaAllowlist("120363abc@g.us"),
		Repo:     repo,
		Canal:    canal,
		Midias:   &memMidia{files: map[string][]byte{}},
		AckTexto: "ack-teste",
		Agente:   ag,
		NewID:    seqIDs(),
	})
	// 1. O gateway envia uma resposta no grupo
	msg, err := gw.Enviar(context.Background(), application.Saida{
		ConversaJID: "120363abc@g.us",
		Corpo:       "Tomate: R$ 5,99",
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg.ProvedorID == "" {
		t.Fatal("esperava ProvedorID preenchido após envio")
	}

	// 2. O WhatsApp ecoa a mensagem enviada de volta para o gateway
	_, err = gw.Receber(context.Background(), application.Entrada{
		ProvedorID:   msg.ProvedorID,
		ConversaJID:  "120363abc@g.us",
		RemetenteJID: "5511999999999",
		Grupo:        true,
		FromMe:       true,
		Corpo:        "Tomate: R$ 5,99",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 3. Garante que o eco não re-dispara o Agente (prevenção de loop infinito)
	if len(ag.calls) != 0 {
		t.Fatalf("eco da mensagem do próprio bot disparou agente: %+v", ag.calls)
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
	gw     *application.Gateway
	repo   *memRepo
	canal  *stubCanal
	midias *memMidia
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
	return fixture{gw: gw, repo: repo, canal: canal, midias: midias}
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

type stubAgente struct {
	calls []struct{ jid, corpo string }
}

func (s *stubAgente) Atender(_ context.Context, conversaJID, corpo string) error {
	s.calls = append(s.calls, struct{ jid, corpo string }{conversaJID, corpo})
	return nil
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
	if existing, ok := m.lookupContato(c.JID, c.JIDLID); ok {
		return m.mergeContato(existing, c), nil
	}
	m.contatos[c.ID] = c
	m.indexContato(c)
	return c, nil
}

func (m *memRepo) UpsertConversa(_ context.Context, c domain.Conversa) (domain.Conversa, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.lookupConversa(c.JID, c.JIDLID); ok {
		return m.mergeConversa(existing, c), nil
	}
	m.conversas[c.ID] = c
	m.indexConversa(c)
	return c, nil
}

func (m *memRepo) lookupContato(keys ...domain.JID) (domain.Contato, bool) {
	for _, k := range keys {
		if k == "" {
			continue
		}
		if id, ok := m.contatoJID[k]; ok {
			return m.contatos[id], true
		}
	}
	return domain.Contato{}, false
}

func (m *memRepo) lookupConversa(keys ...domain.JID) (domain.Conversa, bool) {
	for _, k := range keys {
		if k == "" {
			continue
		}
		if id, ok := m.convJID[k]; ok {
			return m.conversas[id], true
		}
	}
	return domain.Conversa{}, false
}

func (m *memRepo) indexContato(c domain.Contato) {
	if c.JID != "" {
		m.contatoJID[c.JID] = c.ID
	}
	if c.JIDLID != "" {
		m.contatoJID[c.JIDLID] = c.ID
	}
}

func (m *memRepo) indexConversa(c domain.Conversa) {
	if c.JID != "" {
		m.convJID[c.JID] = c.ID
	}
	if c.JIDLID != "" {
		m.convJID[c.JIDLID] = c.ID
	}
}

func (m *memRepo) mergeContato(existing, in domain.Contato) domain.Contato {
	if existing.JIDLID == "" && in.JIDLID != "" {
		existing.JIDLID = in.JIDLID
	}
	if strings.HasSuffix(string(existing.JID), "@lid") && in.JID != "" && !strings.HasSuffix(string(in.JID), "@lid") {
		existing.JID = in.JID
	}
	m.contatos[existing.ID] = existing
	m.indexContato(existing)
	return existing
}

func (m *memRepo) mergeConversa(existing, in domain.Conversa) domain.Conversa {
	if existing.JIDLID == "" && in.JIDLID != "" {
		existing.JIDLID = in.JIDLID
	}
	if strings.HasSuffix(string(existing.JID), "@lid") && in.JID != "" && !strings.HasSuffix(string(in.JID), "@lid") {
		existing.JID = in.JID
	}
	m.conversas[existing.ID] = existing
	m.indexConversa(existing)
	return existing
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

func (m *memRepo) ListarConversas(_ context.Context) ([]domain.ConversaResumo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.ConversaResumo, 0, len(m.conversas))
	for _, c := range m.conversas {
		r := domain.ConversaResumo{Conversa: c}
		for _, msg := range m.msgs {
			if msg.ConversaID != c.ID {
				continue
			}
			r.TotalMensagens++
			cp := msg
			if r.UltimaMensagem == nil ||
				msg.CriadoEm.After(r.UltimaMensagem.CriadoEm) ||
				(msg.CriadoEm.Equal(r.UltimaMensagem.CriadoEm) && string(msg.ID) > string(r.UltimaMensagem.ID)) {
				r.UltimaMensagem = &cp
			}
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		ti, tj := time.Time{}, time.Time{}
		if out[i].UltimaMensagem != nil {
			ti = out[i].UltimaMensagem.CriadoEm
		}
		if out[j].UltimaMensagem != nil {
			tj = out[j].UltimaMensagem.CriadoEm
		}
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return string(out[i].ID) < string(out[j].ID)
	})
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

func (m *memRepo) MarcarRecibo(_ context.Context, provedorID string, status domain.StatusEnvio) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byProv[provedorID]
	if !ok {
		return false, nil
	}
	msg := m.msgs[id]
	msg.Status = status
	m.msgs[id] = msg
	return true, nil
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

func (s *stubCanal) Enviar(_ context.Context, destino domain.JID, corpo string, midia *domain.MidiaBytes) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return "", s.err
	}
	var copyMidia *domain.MidiaBytes
	if midia != nil {
		c := *midia
		c.Conteudo = bytes.Clone(midia.Conteudo)
		copyMidia = &c
	}
	s.envios = append(s.envios, stubEnvio{Destino: destino, Corpo: corpo, Midia: copyMidia})
	return "stub", nil
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
	fail  bool
}

func (m *memMidia) Guardar(_ context.Context, mensagemID domain.MensagemID, midia domain.MidiaBytes) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return "", errors.New("midia falhou")
	}
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
