package whatsapp

import (
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
)

// Inbound is the channel-neutral snapshot of a WhatsApp event (filled by the whatsmeow adapter).
type Inbound struct {
	ProvedorID string
	ChatJID    string
	SenderJID  string
	Grupo      bool
	FromMe     bool
	Texto      string
	Midia      *domain.MidiaBytes
}

func Entrada(in Inbound) (application.Entrada, bool) {
	if in.FromMe {
		return application.Entrada{}, false
	}
	if in.Texto == "" && (in.Midia == nil || len(in.Midia.Conteudo) == 0) {
		return application.Entrada{}, false
	}
	return application.Entrada{
		ProvedorID:   in.ProvedorID,
		ConversaJID:  in.ChatJID,
		RemetenteJID: in.SenderJID,
		Grupo:        in.Grupo,
		Corpo:        in.Texto,
		Midia:        in.Midia,
	}, true
}
