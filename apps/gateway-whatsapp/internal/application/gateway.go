package application

import (
	"context"
	"errors"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
)

var ErrNaoPermitido = errors.New("conversa fora da allowlist")

type Agente interface {
	Atender(ctx context.Context, conversaJID, corpo string) error
}

type Deps struct {
	Allow    domain.Allowlist
	Repo     domain.Repositorio
	Canal    domain.Canal
	Midias   domain.MidiaStore
	AckTexto string
	Agente   Agente
	NewID    func() string
}

type Gateway struct {
	allow    domain.Allowlist
	repo     domain.Repositorio
	canal    domain.Canal
	midias   domain.MidiaStore
	ackTexto string
	agente   Agente
	newID    func() string
	mu       sync.Mutex
	enviadas map[string]time.Time
}

func New(d Deps) *Gateway {
	newID := d.NewID
	if newID == nil {
		newID = func() string { return "" }
	}
	return &Gateway{
		allow:    d.Allow,
		repo:     d.Repo,
		canal:    d.Canal,
		midias:   d.Midias,
		ackTexto: d.AckTexto,
		agente:   d.Agente,
		newID:    newID,
		enviadas: make(map[string]time.Time),
	}
}

type Entrada struct {
	ProvedorID   string
	ConversaJID  string
	ConversaPN   string
	ConversaLID  string
	RemetenteJID string
	RemetentePN  string
	RemetenteLID string
	Grupo        bool
	FromMe       bool
	Status       bool
	Origem       domain.OrigemMensagem
	CriadoEm     time.Time
	PushName     string
	ConversaNome string
	Corpo        string
	Tipo         domain.TipoMensagem
	Payload      string
	Midia        *domain.MidiaBytes
}

type Saida struct {
	ConversaJID string
	Corpo       string
	Midia       *domain.MidiaBytes
}

type ResultadoReceber struct {
	Aceita    bool
	Conversa  domain.Conversa
	Mensagem  domain.Mensagem
	Duplicada bool
}

func (g *Gateway) Receber(ctx context.Context, in Entrada) (ResultadoReceber, error) {
	if in.ProvedorID != "" {
		if existing, ok, err := g.repo.MensagemPorProvedor(ctx, in.ProvedorID); err != nil {
			return ResultadoReceber{}, err
		} else if ok {
			conversa, _, err := g.repo.GetConversa(ctx, existing.ConversaID)
			if err != nil {
				return ResultadoReceber{}, err
			}
			changed := false
			eraVazio := strings.TrimSpace(existing.Corpo) == ""
			if in.Corpo != existing.Corpo {
				existing.Corpo = in.Corpo
				changed = true
			}
			if in.Tipo != "" && in.Tipo != existing.Tipo {
				existing.Tipo = in.Tipo
				changed = true
			}
			if in.Payload != "" && in.Payload != existing.Payload {
				existing.Payload = in.Payload
				changed = true
			}
			if changed {
				if err := g.repo.SalvarMensagem(ctx, existing); err != nil {
					return ResultadoReceber{}, err
				}
			}
			if eraVazio && strings.TrimSpace(in.Corpo) != "" && g.deveAgente(in) {
				log.Printf("whatsapp agente atender (corpo atualizado) conversa=%s corpo=%q", conversa.JID, in.Corpo)
				if err := g.agente.Atender(ctx, string(conversa.JID), in.Corpo); err != nil {
					log.Printf("whatsapp agente atender falhou: %v", err)
				}
			}
			return ResultadoReceber{Aceita: true, Conversa: conversa, Mensagem: existing, Duplicada: true}, nil
		}
	}
	convJID, convLID := domain.Identidade(in.ConversaJID, in.ConversaPN, in.ConversaLID)
	remJID, remLID := domain.Identidade(in.RemetenteJID, in.RemetentePN, in.RemetenteLID)
	if remJID == "" {
		remJID = convJID
		remLID = convLID
	}
	tipo := domain.ConversaDireta
	if in.Status || strings.HasSuffix(string(convJID), "@broadcast") {
		tipo = domain.ConversaStatus
	} else if in.Grupo {
		tipo = domain.ConversaGrupo
	}
	contato, err := g.repo.UpsertContato(ctx, domain.Contato{ID: domain.ContatoID(g.newID()), JID: remJID, JIDLID: remLID})
	if err != nil {
		return ResultadoReceber{}, err
	}
	conversa, err := g.repo.UpsertConversa(ctx, domain.Conversa{ID: domain.ConversaID(g.newID()), JID: convJID, JIDLID: convLID, Tipo: tipo})
	if err != nil {
		return ResultadoReceber{}, err
	}
	msg := domain.Mensagem{
		ID:         domain.MensagemID(g.newID()),
		ConversaID: conversa.ID,
		ContatoID:  contato.ID,
		Direcao:    domain.DirecaoEntrada,
		Corpo:      in.Corpo,
		ProvedorID: in.ProvedorID,
		CriadoEm:   in.CriadoEm.UTC(),
		Origem:     in.Origem,
		PushName:   in.PushName,
		Tipo:       in.Tipo,
		Payload:    in.Payload,
	}
	if msg.CriadoEm.IsZero() {
		msg.CriadoEm = time.Now().UTC()
	}
	if msg.Origem == "" {
		msg.Origem = domain.OrigemVivo
	}
	if in.FromMe {
		msg.Direcao = domain.DirecaoSaida
		msg.Status = domain.StatusEnviado
	}
	if in.Midia != nil {
		if err := g.anexarMidia(ctx, &msg, in.Midia); err != nil {
			return ResultadoReceber{}, err
		}
	}
	if err := g.repo.SalvarMensagem(ctx, msg); err != nil {
		if errors.Is(err, domain.ErrProvedorDuplicado) && in.ProvedorID != "" {
			return g.receberDuplicada(ctx, in)
		}
		return ResultadoReceber{}, err
	}
	if in.Grupo && !g.allow.PermiteConversa(in.ConversaJID, in.ConversaPN, in.ConversaLID) {
		log.Printf("whatsapp grupo fora da allowlist: jid=%s nome=%q (adicione à WHATSAPP_ALLOWLIST para responder)", in.ConversaJID, in.ConversaNome)
	}
	if g.deveAgente(in) {
		log.Printf("whatsapp agente atender conversa=%s corpo=%q", conversa.JID, in.Corpo)
		if err := g.agente.Atender(ctx, string(conversa.JID), in.Corpo); err != nil {
			log.Printf("whatsapp agente atender falhou: %v", err)
		}
	} else if g.deveAck(in) {
		log.Printf("whatsapp ack enviado conversa=%s", conversa.JID)
		_, _ = g.enviarNaConversa(ctx, conversa, contato, g.ackTexto, nil)
	}
	return ResultadoReceber{Aceita: true, Conversa: conversa, Mensagem: msg}, nil
}

func (g *Gateway) registrarEnviada(provedorID string) {
	if provedorID == "" {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.enviadas == nil {
		g.enviadas = make(map[string]time.Time)
	}
	agora := time.Now()
	g.enviadas[provedorID] = agora
	for id, t := range g.enviadas {
		if agora.Sub(t) > 10*time.Minute {
			delete(g.enviadas, id)
		}
	}
}

func (g *Gateway) foiEnviada(provedorID string) bool {
	if provedorID == "" {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.enviadas == nil {
		return false
	}
	_, ok := g.enviadas[provedorID]
	return ok
}

func (g *Gateway) receberDuplicada(ctx context.Context, in Entrada) (ResultadoReceber, error) {
	existing, ok, err := g.repo.MensagemPorProvedor(ctx, in.ProvedorID)
	if err != nil {
		return ResultadoReceber{}, err
	}
	if !ok {
		return ResultadoReceber{}, domain.ErrProvedorDuplicado
	}
	conversa, _, err := g.repo.GetConversa(ctx, existing.ConversaID)
	if err != nil {
		return ResultadoReceber{}, err
	}
	return ResultadoReceber{Aceita: true, Conversa: conversa, Mensagem: existing, Duplicada: true}, nil
}

func (g *Gateway) deveAck(in Entrada) bool {
	if g.agente != nil || g.ackTexto == "" || in.Status || in.Origem == domain.OrigemHistorico {
		return false
	}
	switch in.Tipo {
	case domain.MensagemReacao, domain.MensagemRevogacao, domain.MensagemIndecifravel:
		return false
	}
	if strings.TrimSpace(in.Corpo) == "" {
		return false
	}
	if in.FromMe {
		if !in.Grupo || g.foiEnviada(in.ProvedorID) {
			return false
		}
	}
	return g.allow.PermiteConversa(in.ConversaJID, in.ConversaPN, in.ConversaLID)
}

func (g *Gateway) deveAgente(in Entrada) bool {
	if g.agente == nil || strings.TrimSpace(in.Corpo) == "" {
		return false
	}
	switch in.Tipo {
	case domain.MensagemReacao, domain.MensagemRevogacao, domain.MensagemIndecifravel:
		return false
	}
	if in.Status || in.Origem == domain.OrigemHistorico {
		return false
	}
	if in.FromMe {
		if !in.Grupo || g.foiEnviada(in.ProvedorID) {
			return false
		}
	}
	return g.allow.PermiteConversa(in.ConversaJID, in.ConversaPN, in.ConversaLID)
}

func (g *Gateway) Enviar(ctx context.Context, out Saida) (domain.Mensagem, error) {
	if !g.allow.PermiteConversa(out.ConversaJID) {
		return domain.Mensagem{}, ErrNaoPermitido
	}
	jid := domain.NormalizarJID(out.ConversaJID)
	tipo := domain.ConversaDireta
	if strings.HasSuffix(string(jid), "@g.us") {
		tipo = domain.ConversaGrupo
	}
	contato, err := g.repo.UpsertContato(ctx, domain.Contato{ID: domain.ContatoID(g.newID()), JID: jid})
	if err != nil {
		return domain.Mensagem{}, err
	}
	conversa, err := g.repo.UpsertConversa(ctx, domain.Conversa{ID: domain.ConversaID(g.newID()), JID: jid, Tipo: tipo})
	if err != nil {
		return domain.Mensagem{}, err
	}
	return g.enviarNaConversa(ctx, conversa, contato, out.Corpo, out.Midia)
}

func (g *Gateway) enviarNaConversa(ctx context.Context, conversa domain.Conversa, contato domain.Contato, corpo string, midia *domain.MidiaBytes) (domain.Mensagem, error) {
	msg := domain.Mensagem{
		ID:         domain.MensagemID(g.newID()),
		ConversaID: conversa.ID,
		ContatoID:  contato.ID,
		Direcao:    domain.DirecaoSaida,
		Corpo:      corpo,
		Status:     domain.StatusPendente,
		Origem:     domain.OrigemVivo,
		CriadoEm:   time.Now().UTC(),
	}
	if midia != nil {
		if err := g.anexarMidia(ctx, &msg, midia); err != nil {
			return domain.Mensagem{}, err
		}
	}
	if err := g.repo.SalvarMensagem(ctx, msg); err != nil {
		return domain.Mensagem{}, err
	}
	var canalMidia *domain.MidiaBytes
	if midia != nil {
		canalMidia = midia
	}
	if g.canal != nil {
		id, err := g.canal.Enviar(ctx, conversa.JID, corpo, canalMidia)
		if err != nil {
			msg.Status = domain.StatusFalhou
			_ = g.repo.SalvarMensagem(ctx, msg)
			return msg, err
		}
		if id != "" {
			msg.ProvedorID = id
			g.registrarEnviada(id)
		}
	}
	msg.Status = domain.StatusEnviado
	if err := g.repo.SalvarMensagem(ctx, msg); err != nil {
		return msg, err
	}
	return msg, nil
}

func (g *Gateway) anexarMidia(ctx context.Context, msg *domain.Mensagem, midia *domain.MidiaBytes) error {
	msg.Midia = &domain.Midia{Tipo: midia.Tipo, Filename: midia.Filename, MIME: midia.MIME}
	if g.midias == nil {
		return nil
	}
	path, err := g.midias.Guardar(ctx, msg.ID, *midia)
	if err != nil {
		return nil
	}
	msg.Midia.Path = path
	return nil
}

func (g *Gateway) MarcarRecibo(ctx context.Context, provedorID string, status domain.StatusEnvio) error {
	if provedorID == "" || status == "" {
		return nil
	}
	_, err := g.repo.MarcarRecibo(ctx, provedorID, status)
	return err
}

type FiltroConversas struct {
	Tipo domain.TipoConversa
	Q    string
}

type ConversaResumo struct {
	domain.ConversaResumo
	Permitido bool
}

func (g *Gateway) ListarMensagens(ctx context.Context, conversaID domain.ConversaID) ([]domain.Mensagem, error) {
	return g.repo.ListarMensagens(ctx, conversaID)
}

func (g *Gateway) ListarConversas(ctx context.Context, f FiltroConversas) ([]ConversaResumo, error) {
	items, err := g.repo.ListarConversas(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ConversaResumo, 0, len(items))
	q := strings.ToLower(strings.TrimSpace(f.Q))
	for _, item := range items {
		if f.Tipo != "" && item.Tipo != f.Tipo {
			continue
		}
		if q != "" && !conversaCasaQ(item, q) {
			continue
		}
		out = append(out, ConversaResumo{
			ConversaResumo: item,
			Permitido:      g.allow.PermiteConversa(string(item.JID), string(item.JIDLID)),
		})
	}
	return out, nil
}

func conversaCasaQ(item domain.ConversaResumo, q string) bool {
	if strings.Contains(strings.ToLower(string(item.JID)), q) {
		return true
	}
	if strings.Contains(strings.ToLower(string(item.JIDLID)), q) {
		return true
	}
	if item.UltimaMensagem != nil && strings.Contains(strings.ToLower(item.UltimaMensagem.PushName), q) {
		return true
	}
	return false
}

func (g *Gateway) ObterConversa(ctx context.Context, id domain.ConversaID) (ConversaResumo, bool, error) {
	items, err := g.ListarConversas(ctx, FiltroConversas{})
	if err != nil {
		return ConversaResumo{}, false, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, true, nil
		}
	}
	return ConversaResumo{}, false, nil
}

func (g *Gateway) ObterMensagem(ctx context.Context, id domain.MensagemID) (domain.Mensagem, bool, error) {
	return g.repo.GetMensagem(ctx, id)
}

func (g *Gateway) ListarMensagensPagina(ctx context.Context, conversaID domain.ConversaID, limit int, antesCriado time.Time, antesID domain.MensagemID) ([]domain.Mensagem, error) {
	if limit <= 0 {
		limit = 200
	}
	all, err := g.repo.ListarMensagens(ctx, conversaID)
	if err != nil {
		return nil, err
	}
	sort.Slice(all, func(i, j int) bool {
		if !all[i].CriadoEm.Equal(all[j].CriadoEm) {
			return all[i].CriadoEm.After(all[j].CriadoEm)
		}
		return string(all[i].ID) > string(all[j].ID)
	})
	out := make([]domain.Mensagem, 0, limit)
	for _, m := range all {
		if !antesCriado.IsZero() || antesID != "" {
			if m.CriadoEm.After(antesCriado) || (m.CriadoEm.Equal(antesCriado) && string(m.ID) >= string(antesID)) {
				continue
			}
		}
		out = append(out, m)
		if len(out) == limit {
			break
		}
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (g *Gateway) LerMidia(ctx context.Context, path string) ([]byte, error) {
	return g.midias.Ler(ctx, path)
}

func (g *Gateway) Conectado() bool {
	if g.canal == nil {
		return false
	}
	return g.canal.Conectado()
}
