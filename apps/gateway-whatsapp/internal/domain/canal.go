package domain

type EstadoCanal string

const (
	CanalNaoConfigurado EstadoCanal = "nao_configurado"
	CanalPronto         EstadoCanal = "pronto"
)

// CanalSituacao is the operator-facing snapshot of the Canal.
type CanalSituacao struct {
	Estado EstadoCanal
	JID    JID
}

type Template struct {
	Nome   string
	Idioma string
}

type Envio struct {
	Destino  JID
	Corpo    string
	Midia    *MidiaBytes
	Template *Template
}
