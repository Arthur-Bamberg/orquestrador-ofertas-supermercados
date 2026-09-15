package domain

import "context"

type MidiaBytes struct {
	Tipo     TipoMidia
	Filename string
	MIME     string
	Conteudo []byte
}

type Canal interface {
	Enviar(ctx context.Context, e Envio) (provedorID string, err error)
	Pronto() bool
	Situacao() CanalSituacao
}

type Repositorio interface {
	UpsertContato(ctx context.Context, c Contato) (Contato, error)
	UpsertConversa(ctx context.Context, c Conversa) (Conversa, error)
	SalvarMensagem(ctx context.Context, m Mensagem) error
	MensagemPorProvedor(ctx context.Context, provedorID string) (Mensagem, bool, error)
	ListarMensagens(ctx context.Context, conversaID ConversaID) ([]Mensagem, error)
	ListarConversas(ctx context.Context) ([]ConversaResumo, error)
	GetMensagem(ctx context.Context, id MensagemID) (Mensagem, bool, error)
	GetConversa(ctx context.Context, id ConversaID) (Conversa, bool, error)
	ConversaPorJID(ctx context.Context, jid JID) (Conversa, bool, error)
	MarcarRecibo(ctx context.Context, provedorID string, status StatusEnvio) (bool, error)
}

type MidiaStore interface {
	Guardar(ctx context.Context, mensagemID MensagemID, midia MidiaBytes) (path string, err error)
	Ler(ctx context.Context, path string) ([]byte, error)
}
