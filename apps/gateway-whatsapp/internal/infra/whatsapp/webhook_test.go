package whatsapp_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/whatsapp"
)

func TestParseWebhook_textoDireta(t *testing.T) {
	body := []byte(`{
		"object":"whatsapp_business_account",
		"entry":[{"changes":[{"value":{
			"contacts":[{"profile":{"name":"Ana"},"wa_id":"5511999999999"}],
			"messages":[{"from":"5511999999999","id":"wamid.1","timestamp":"1757800000","type":"text","text":{"body":"tomate"}}]
		}}]}]
	}`)
	msgs, recs, err := whatsapp.ParseWebhook(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 0 || len(msgs) != 1 {
		t.Fatalf("msgs=%d recs=%d", len(msgs), len(recs))
	}
	got := msgs[0]
	if got.ProvedorID != "wamid.1" || got.Texto != "tomate" || got.PushName != "Ana" {
		t.Fatalf("%+v", got)
	}
	if got.ChatJID != "5511999999999@s.whatsapp.net" || got.Grupo || got.Status {
		t.Fatalf("jid/grupo %+v", got)
	}
	if !got.CriadoEm.Equal(time.Unix(1757800000, 0).UTC()) {
		t.Fatalf("ts %s", got.CriadoEm)
	}
	in, ok := whatsapp.Entrada(got)
	if !ok || in.Corpo != "tomate" || in.Grupo {
		t.Fatalf("entrada %+v ok=%v", in, ok)
	}
}

func TestParseWebhook_ignoraGrupo(t *testing.T) {
	body := []byte(`{
		"entry":[{"changes":[{"value":{"messages":[
			{"from":"5511999999999","id":"wamid.g","timestamp":"1","type":"text","text":{"body":"oi"},"group_id":"123"}
		]}}]}]
	}`)
	msgs, _, err := whatsapp.ParseWebhook(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Fatalf("grupo ingerido %+v", msgs)
	}
}

func TestParseWebhook_imagemERecibo(t *testing.T) {
	body := []byte(`{
		"entry":[{"changes":[{"value":{
			"messages":[{"from":"5511888888888","id":"wamid.img","timestamp":"10","type":"image","image":{"id":"media-1","mime_type":"image/jpeg","caption":"foto"}}],
			"statuses":[{"id":"wamid.out","status":"delivered"}]
		}}]}]
	}`)
	msgs, recs, err := whatsapp.ParseWebhook(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].MidiaID != "media-1" || msgs[0].Texto != "foto" || msgs[0].Midia == nil || msgs[0].Midia.Tipo != domain.MidiaImagem {
		t.Fatalf("msgs %+v", msgs)
	}
	if len(recs) != 1 || recs[0].ProvedorID != "wamid.out" || recs[0].Status != domain.StatusEntregue {
		t.Fatalf("recs %+v", recs)
	}
}

func TestParseWebhook_reacao(t *testing.T) {
	body := []byte(`{
		"entry":[{"changes":[{"value":{"messages":[
			{"from":"5511999999999","id":"wamid.r","timestamp":"1","type":"reaction","reaction":{"message_id":"alvo","emoji":"👍"}}
		]}}]}]
	}`)
	msgs, _, err := whatsapp.ParseWebhook(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Tipo != domain.MensagemReacao || msgs[0].Texto != "👍" {
		t.Fatalf("%+v", msgs)
	}
}

func TestAssinaturaValida(t *testing.T) {
	body := []byte(`{"object":"whatsapp_business_account"}`)
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(body)
	header := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !whatsapp.AssinaturaValida("secret", header, body) {
		t.Fatal("válida")
	}
	if whatsapp.AssinaturaValida("secret", "sha256=deadbeef", body) {
		t.Fatal("inválida aceita")
	}
	if !whatsapp.AssinaturaValida("", "qualquer", body) {
		t.Fatal("sem secret deve aceitar")
	}
}
