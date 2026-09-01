package domain

import "time"

type ContatoID string
type ConversaID string
type MensagemID string

type TipoConversa string

const (
	ConversaDireta TipoConversa = "direta"
	ConversaGrupo  TipoConversa = "grupo"
	ConversaStatus TipoConversa = "status"
)

type Direcao string

const (
	DirecaoEntrada Direcao = "entrada"
	DirecaoSaida   Direcao = "saida"
)

type StatusEnvio string

const (
	StatusPendente    StatusEnvio = "pendente"
	StatusEnviado     StatusEnvio = "enviado"
	StatusFalhou      StatusEnvio = "falhou"
	StatusEntregue    StatusEnvio = "entregue"
	StatusLido        StatusEnvio = "lido"
	StatusReproduzido StatusEnvio = "reproduzido"
)

type TipoMensagem string

const (
	MensagemTexto        TipoMensagem = "texto"
	MensagemMidia        TipoMensagem = "midia"
	MensagemReacao       TipoMensagem = "reacao"
	MensagemRevogacao    TipoMensagem = "revogacao"
	MensagemIndecifravel TipoMensagem = "indecifravel"
)

type OrigemMensagem string

const (
	OrigemVivo      OrigemMensagem = "vivo"
	OrigemHistorico OrigemMensagem = "historico"
)

type TipoMidia string

const (
	MidiaImagem    TipoMidia = "imagem"
	MidiaAudio     TipoMidia = "audio"
	MidiaVideo     TipoMidia = "video"
	MidiaDocumento TipoMidia = "documento"
	MidiaFigurinha TipoMidia = "figurinha"
)

type Contato struct {
	ID     ContatoID
	JID    JID
	JIDLID JID
}

type Conversa struct {
	ID     ConversaID
	JID    JID
	JIDLID JID
	Tipo   TipoConversa
}

type ConversaResumo struct {
	Conversa
	UltimaMensagem *Mensagem
	TotalMensagens int
}

type Midia struct {
	Tipo     TipoMidia
	Path     string
	Filename string
	MIME     string
}

type Mensagem struct {
	ID         MensagemID
	ConversaID ConversaID
	ContatoID  ContatoID
	Direcao    Direcao
	Corpo      string
	Midia      *Midia
	ProvedorID string
	Status     StatusEnvio
	CriadoEm   time.Time
	Origem     OrigemMensagem
	PushName   string
	Tipo       TipoMensagem
	Payload    string
}
