package domain

type EstadoCanal string

const (
	CanalPendente     EstadoCanal = "pendente"
	CanalConectado    EstadoCanal = "conectado"
	CanalDesconectado EstadoCanal = "desconectado"
)

// CanalSituacao is the operator-facing snapshot of the Canal.
type CanalSituacao struct {
	Estado EstadoCanal
	JID    JID
	QR     string
}
