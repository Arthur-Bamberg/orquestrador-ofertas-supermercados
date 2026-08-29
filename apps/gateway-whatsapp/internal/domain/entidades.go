package domain

import "time"

type ContatoID string
type ConversaID string
type MensagemID string

type TipoConversa string

const (
	ConversaDireta TipoConversa = "direta"
	ConversaGrupo  TipoConversa = "grupo"
)

type Direcao string

const (
	DirecaoEntrada Direcao = "entrada"
	DirecaoSaida   Direcao = "saida"
)

type StatusEnvio string

const (
	StatusPendente StatusEnvio = "pendente"
	StatusEnviado  StatusEnvio = "enviado"
	StatusFalhou   StatusEnvio = "falhou"
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
	ID  ContatoID
	JID JID
}

type Conversa struct {
	ID   ConversaID
	JID  JID
	Tipo TipoConversa
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
}
