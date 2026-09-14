package whatsapp

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
)

// Inbound is the channel-neutral snapshot of a WhatsApp event (filled by the Cloud API adapter).
type Inbound struct {
	ProvedorID   string
	ChatJID      string
	ChatPN       string
	ChatLID      string
	SenderJID    string
	SenderPN     string
	SenderLID    string
	Grupo        bool
	Status       bool
	FromMe       bool
	Texto        string
	PushName     string
	ConversaNome string
	CriadoEm     time.Time
	Origem       domain.OrigemMensagem
	Tipo         domain.TipoMensagem
	Payload      string
	Midia        *domain.MidiaBytes
	MidiaID      string
}

func Entrada(in Inbound) (application.Entrada, bool) {
	if !temConteudo(in) {
		return application.Entrada{}, false
	}
	origem := in.Origem
	if origem == "" {
		origem = domain.OrigemVivo
	}
	return application.Entrada{
		ProvedorID:   in.ProvedorID,
		ConversaJID:  in.ChatJID,
		ConversaPN:   in.ChatPN,
		ConversaLID:  in.ChatLID,
		RemetenteJID: in.SenderJID,
		RemetentePN:  in.SenderPN,
		RemetenteLID: in.SenderLID,
		Grupo:        in.Grupo,
		Status:       in.Status || strings.HasSuffix(strings.ToLower(in.ChatJID), "@broadcast"),
		FromMe:       in.FromMe,
		Corpo:        in.Texto,
		Midia:        in.Midia,
		PushName:     in.PushName,
		ConversaNome: in.ConversaNome,
		CriadoEm:     in.CriadoEm,
		Origem:       origem,
		Tipo:         in.Tipo,
		Payload:      in.Payload,
	}, true
}

func temConteudo(in Inbound) bool {
	if strings.TrimSpace(in.Texto) != "" {
		return true
	}
	if in.Midia != nil && (len(in.Midia.Conteudo) > 0 || in.Midia.Tipo != "") {
		return true
	}
	switch in.Tipo {
	case domain.MensagemReacao, domain.MensagemRevogacao, domain.MensagemIndecifravel:
		return true
	}
	return false
}

func payloadJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
