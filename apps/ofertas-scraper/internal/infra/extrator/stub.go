package extrator

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

// Stub returns fixed candidatos (ADR 0023).
type Stub struct {
	Candidatos []domain.CandidatoOferta
	Raw        []byte
	Err        error
}

func (s Stub) Extract(context.Context, []domain.PageImage) ([]domain.CandidatoOferta, []byte, error) {
	if s.Err != nil {
		return nil, nil, s.Err
	}
	raw := s.Raw
	if raw == nil {
		payload := map[string]any{"ofertas": s.Candidatos}
		var err error
		raw, err = json.Marshal(payload)
		if err != nil {
			return nil, nil, err
		}
	}
	return s.Candidatos, raw, nil
}

// Unavailable always returns ErrExtratorIndisponivel.
func Unavailable() Stub {
	return Stub{Err: domain.ErrExtratorIndisponivel}
}

// IsIndisponivel reports whether err (or a wrapped err) means Extrator outage.
func IsIndisponivel(err error) bool {
	return errors.Is(err, domain.ErrExtratorIndisponivel)
}
