package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
)

var ErrNaoPermitido = errors.New("conversa fora da allowlist")

type Deps struct {
	Allow    domain.Allowlist
	Repo     domain.Repositorio
	Canal    domain.Canal
	Midias   domain.MidiaStore
	AckTexto string
	NewID    func() string
}

type Gateway struct {
	allow    domain.Allowlist
	repo     domain.Repositorio
	canal    domain.Canal
	midias   domain.MidiaStore
	ackTexto string
	newID    func() string
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
		newID:    newID,
	}
}

type Entrada struct {
	ProvedorID   string
	ConversaJID  string
	RemetenteJID string
	Grupo        bool
	Corpo        string
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
	if !g.allow.PermiteConversa(in.ConversaJID) {
		return ResultadoReceber{}, nil
	}
	if in.ProvedorID != "" {
		if existing, ok, err := g.repo.MensagemPorProvedor(ctx, in.ProvedorID); err != nil {
			return ResultadoReceber{}, err
		} else if ok {
			conversa, _, err := g.repo.GetConversa(ctx, existing.ConversaID)
			if err != nil {
				return ResultadoReceber{}, err
			}
			return ResultadoReceber{Aceita: true, Conversa: conversa, Mensagem: existing, Duplicada: true}, nil
		}
	}
	convJID := domain.NormalizarJID(in.ConversaJID)
	remJID := domain.NormalizarJID(in.RemetenteJID)
	if remJID == "" {
		remJID = convJID
	}
	tipo := domain.ConversaDireta
	if in.Grupo {
		tipo = domain.ConversaGrupo
	}
	contato, err := g.repo.UpsertContato(ctx, domain.Contato{ID: domain.ContatoID(g.newID()), JID: remJID})
	if err != nil {
		return ResultadoReceber{}, err
	}
	conversa, err := g.repo.UpsertConversa(ctx, domain.Conversa{ID: domain.ConversaID(g.newID()), JID: convJID, Tipo: tipo})
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
		CriadoEm:   time.Now().UTC(),
	}
	if in.Midia != nil {
		if err := g.anexarMidia(ctx, &msg, in.Midia); err != nil {
			return ResultadoReceber{}, err
		}
	}
	if err := g.repo.SalvarMensagem(ctx, msg); err != nil {
		return ResultadoReceber{}, err
	}
	if g.ackTexto != "" {
		_, _ = g.enviarNaConversa(ctx, conversa, contato, g.ackTexto, nil)
	}
	return ResultadoReceber{Aceita: true, Conversa: conversa, Mensagem: msg}, nil
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
		if err := g.canal.Enviar(ctx, conversa.JID, corpo, canalMidia); err != nil {
			msg.Status = domain.StatusFalhou
			_ = g.repo.SalvarMensagem(ctx, msg)
			return msg, err
		}
	}
	msg.Status = domain.StatusEnviado
	if err := g.repo.SalvarMensagem(ctx, msg); err != nil {
		return msg, err
	}
	return msg, nil
}

func (g *Gateway) anexarMidia(ctx context.Context, msg *domain.Mensagem, midia *domain.MidiaBytes) error {
	if g.midias == nil {
		return nil
	}
	path, err := g.midias.Guardar(ctx, msg.ID, *midia)
	if err != nil {
		return err
	}
	msg.Midia = &domain.Midia{
		Tipo:     midia.Tipo,
		Path:     path,
		Filename: midia.Filename,
		MIME:     midia.MIME,
	}
	return nil
}

func (g *Gateway) ListarMensagens(ctx context.Context, conversaID domain.ConversaID) ([]domain.Mensagem, error) {
	return g.repo.ListarMensagens(ctx, conversaID)
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
