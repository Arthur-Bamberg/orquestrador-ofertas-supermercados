package domain_test

import (
	"testing"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
)

func TestDentroDaJanela(t *testing.T) {
	agora := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	casos := []struct {
		nome   string
		ultima time.Time
		dentro bool
	}{
		{"sem entrada", time.Time{}, false},
		{"agora", agora, true},
		{"23h", agora.Add(-23 * time.Hour), true},
		{"24h", agora.Add(-24 * time.Hour), true},
		{"24h1s", agora.Add(-24*time.Hour - time.Second), false},
		{"relógio atrasado", agora.Add(time.Minute), true},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := domain.DentroDaJanela(c.ultima, agora); got != c.dentro {
				t.Fatalf("got %v want %v", got, c.dentro)
			}
		})
	}
}
