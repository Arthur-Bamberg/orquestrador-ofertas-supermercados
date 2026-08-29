package canal

import (
	"context"
	"sync"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
)

type Stub struct {
	mu     sync.Mutex
	Envios []Envio
}

type Envio struct {
	Destino domain.JID
	Corpo   string
	Midia   *domain.MidiaBytes
}

func (s *Stub) Enviar(_ context.Context, destino domain.JID, corpo string, midia *domain.MidiaBytes) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Envios = append(s.Envios, Envio{Destino: destino, Corpo: corpo, Midia: midia})
	return nil
}

func (s *Stub) Conectado() bool { return true }
