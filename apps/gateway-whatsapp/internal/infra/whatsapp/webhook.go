package whatsapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
)

type Recibo struct {
	ProvedorID string
	Status     domain.StatusEnvio
}

type webhookPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		Changes []struct {
			Value struct {
				Contacts []struct {
					WaID    string `json:"wa_id"`
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
				} `json:"contacts"`
				Messages []msgJSON    `json:"messages"`
				Statuses []statusJSON `json:"statuses"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

type msgJSON struct {
	From      string `json:"from"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	GroupID   string `json:"group_id"`
	Text      *struct {
		Body string `json:"body"`
	} `json:"text"`
	Image    *mediaJSON `json:"image"`
	Audio    *mediaJSON `json:"audio"`
	Video    *mediaJSON `json:"video"`
	Document *mediaJSON `json:"document"`
	Sticker  *mediaJSON `json:"sticker"`
	Reaction *struct {
		MessageID string `json:"message_id"`
		Emoji     string `json:"emoji"`
	} `json:"reaction"`
	Context *struct {
		GroupID string `json:"group_id"`
	} `json:"context"`
}

type mediaJSON struct {
	ID       string `json:"id"`
	Mime     string `json:"mime_type"`
	Caption  string `json:"caption"`
	Filename string `json:"filename"`
}

type statusJSON struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func ParseWebhook(body []byte) (mensagens []Inbound, recibos []Recibo, err error) {
	var p webhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, nil, err
	}
	for _, entry := range p.Entry {
		for _, ch := range entry.Changes {
			names := map[string]string{}
			for _, c := range ch.Value.Contacts {
				names[c.WaID] = c.Profile.Name
			}
			for _, m := range ch.Value.Messages {
				in, ok := inboundDe(m, names)
				if !ok {
					continue
				}
				mensagens = append(mensagens, in)
			}
			for _, s := range ch.Value.Statuses {
				if rec, ok := reciboDe(s); ok {
					recibos = append(recibos, rec)
				}
			}
		}
	}
	return mensagens, recibos, nil
}

func inboundDe(m msgJSON, names map[string]string) (Inbound, bool) {
	if m.GroupID != "" || (m.Context != nil && m.Context.GroupID != "") {
		return Inbound{}, false
	}
	from := strings.TrimSpace(m.From)
	if from == "" || strings.Contains(from, "@g.us") || strings.Contains(from, "@broadcast") {
		return Inbound{}, false
	}
	jid := string(domain.NormalizarJID(from))
	in := Inbound{
		ProvedorID: m.ID,
		ChatJID:    jid,
		ChatPN:     jid,
		SenderJID:  jid,
		SenderPN:   jid,
		PushName:   names[from],
		CriadoEm:   unixTime(m.Timestamp),
		Origem:     domain.OrigemVivo,
		Tipo:       domain.MensagemTexto,
	}
	switch m.Type {
	case "text":
		if m.Text != nil {
			in.Texto = m.Text.Body
		}
	case "image":
		in.Tipo = domain.MensagemMidia
		in.Midia, in.MidiaID, in.Texto = midiaDe(m.Image, domain.MidiaImagem, "imagem")
	case "audio":
		in.Tipo = domain.MensagemMidia
		in.Midia, in.MidiaID, in.Texto = midiaDe(m.Audio, domain.MidiaAudio, "audio")
	case "video":
		in.Tipo = domain.MensagemMidia
		in.Midia, in.MidiaID, in.Texto = midiaDe(m.Video, domain.MidiaVideo, "video")
	case "document":
		in.Tipo = domain.MensagemMidia
		in.Midia, in.MidiaID, in.Texto = midiaDe(m.Document, domain.MidiaDocumento, "documento")
	case "sticker":
		in.Tipo = domain.MensagemMidia
		in.Midia, in.MidiaID, _ = midiaDe(m.Sticker, domain.MidiaFigurinha, "figurinha")
	case "reaction":
		in.Tipo = domain.MensagemReacao
		if m.Reaction != nil {
			in.Texto = m.Reaction.Emoji
			in.Payload = payloadJSON(map[string]string{"alvo": m.Reaction.MessageID})
		}
	default:
		return Inbound{}, false
	}
	if !temConteudo(in) && in.MidiaID == "" {
		return Inbound{}, false
	}
	return in, true
}

func midiaDe(m *mediaJSON, tipo domain.TipoMidia, fallback string) (*domain.MidiaBytes, string, string) {
	if m == nil {
		return nil, "", ""
	}
	name := m.Filename
	if name == "" {
		name = fallback
	}
	return &domain.MidiaBytes{Tipo: tipo, MIME: m.Mime, Filename: name}, m.ID, m.Caption
}

func reciboDe(s statusJSON) (Recibo, bool) {
	st := statusCloud(s.Status)
	if st == "" || s.ID == "" {
		return Recibo{}, false
	}
	return Recibo{ProvedorID: s.ID, Status: st}, true
}

func statusCloud(s string) domain.StatusEnvio {
	switch strings.ToLower(s) {
	case "sent":
		return domain.StatusEnviado
	case "delivered":
		return domain.StatusEntregue
	case "read":
		return domain.StatusLido
	case "failed":
		return domain.StatusFalhou
	default:
		return ""
	}
}

func unixTime(raw string) time.Time {
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || n <= 0 {
		return time.Time{}
	}
	return time.Unix(n, 0).UTC()
}

func AssinaturaValida(secret, header string, body []byte) bool {
	if secret == "" {
		return true
	}
	got, ok := strings.CutPrefix(header, "sha256=")
	if !ok || got == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	want := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(got)), []byte(want))
}
