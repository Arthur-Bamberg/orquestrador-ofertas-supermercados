package whatsapp

import (
	"strings"
	"sync"
)

// assuntosGrupo lembra o assunto de grupos vistos no adapter (history, JoinedGroup, GroupInfo).
type assuntosGrupo struct {
	mu   sync.Mutex
	byID map[string]string
}

func (a *assuntosGrupo) lembrar(jid, nome string) {
	jid = strings.TrimSpace(jid)
	nome = strings.TrimSpace(nome)
	if jid == "" || nome == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.byID == nil {
		a.byID = map[string]string{}
	}
	a.byID[jid] = nome
}

func (a *assuntosGrupo) nome(jid string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.byID == nil {
		return ""
	}
	return a.byID[jid]
}
