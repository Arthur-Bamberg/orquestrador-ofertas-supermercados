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
	Destino  domain.JID
	Corpo    string
	Midia    *domain.MidiaBytes
	Template *domain.Template
}

func (s *Stub) Enviar(_ context.Context, e domain.Envio) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Envios = append(s.Envios, Envio{Destino: e.Destino, Corpo: e.Corpo, Midia: e.Midia, Template: e.Template})
	return "stub", nil
}

func (s *Stub) Pronto() bool { return true }

func (s *Stub) Situacao() domain.CanalSituacao {
	return domain.CanalSituacao{Estado: domain.CanalPronto}
}
