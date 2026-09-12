package operador

import (
	"context"
	"sync"
)

type Mem struct {
	mu    sync.Mutex
	porID map[string]Operador
}

func NewMem() *Mem {
	return &Mem{porID: map[string]Operador{}}
}

func (m *Mem) Fixar(identificacaoID string, op Operador) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.porID[identificacaoID] = op
}

func (m *Mem) OperadorPorIdentificacao(_ context.Context, identificacaoID string) (Operador, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	op, ok := m.porID[identificacaoID]
	return op, ok, nil
}
