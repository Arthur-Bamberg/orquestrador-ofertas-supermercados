package extrator_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/extrator"
)

type stubExtrator struct {
	err  error
	name string
}

func (s stubExtrator) Extract(context.Context, []domain.PageImage) ([]domain.CandidatoOferta, []byte, *domain.UsoExtrator, error) {
	if s.err != nil {
		return nil, nil, nil, s.err
	}
	return []domain.CandidatoOferta{{Produto: "X", Valor: 1, Quantidades: []float64{1}, Medida: "g",
		DataInicio: "2026-07-01", DataExpiracao: "2026-07-02"}}, []byte(`{"ofertas":[]}`), &domain.UsoExtrator{Provider: s.name, Model: "m"}, nil
}

type memCota struct {
	flags map[string]bool
}

func (m *memCota) key(provider, dia string) string { return provider + "|" + dia }
func (m *memCota) Esgotado(_ context.Context, provider, dia string) (bool, error) {
	return m.flags[m.key(provider, dia)], nil
}
func (m *memCota) MarcarEsgotado(_ context.Context, provider, dia string) error {
	if m.flags == nil {
		m.flags = map[string]bool{}
	}
	m.flags[m.key(provider, dia)] = true
	return nil
}

func TestFailover_PrimaryCotaUsesSecondary(t *testing.T) {
	cota := &memCota{}
	f := &extrator.Failover{
		Primary:           stubExtrator{err: domain.ErrExtratorCota, name: "gemini"},
		Secondary:         stubExtrator{name: "cursor"},
		PrimaryProvider:   domain.ExtratorProviderGemini,
		SecondaryProvider: domain.ExtratorProviderCursor,
		Cota:              cota,
		Location:          time.UTC,
		Clock:             func() time.Time { return time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC) },
	}
	_, _, uso, err := f.Extract(context.Background(), []domain.PageImage{{Page: 1, JPEG: []byte{1}}})
	if err != nil {
		t.Fatal(err)
	}
	if uso.Provider != "cursor" {
		t.Fatalf("provider=%s", uso.Provider)
	}
	ok, _ := cota.Esgotado(context.Background(), domain.ExtratorProviderGemini, "2026-07-20")
	if !ok {
		t.Fatal("gemini should be marked esgotado")
	}
}

func TestFailover_SkipsEsgotadoPrimary(t *testing.T) {
	cota := &memCota{flags: map[string]bool{"gemini|2026-07-20": true}}
	f := &extrator.Failover{
		Primary:           stubExtrator{err: errors.New("should not call"), name: "gemini"},
		Secondary:         stubExtrator{name: "cursor"},
		PrimaryProvider:   domain.ExtratorProviderGemini,
		SecondaryProvider: domain.ExtratorProviderCursor,
		Cota:              cota,
		Location:          time.UTC,
		Clock:             func() time.Time { return time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC) },
	}
	_, _, uso, err := f.Extract(context.Background(), []domain.PageImage{{Page: 1, JPEG: []byte{1}}})
	if err != nil {
		t.Fatal(err)
	}
	if uso.Provider != "cursor" {
		t.Fatalf("provider=%s", uso.Provider)
	}
}

func TestFailover_BothEsgotado(t *testing.T) {
	cota := &memCota{flags: map[string]bool{
		"gemini|2026-07-20": true,
		"cursor|2026-07-20": true,
	}}
	f := &extrator.Failover{
		Primary:           stubExtrator{name: "gemini"},
		Secondary:         stubExtrator{name: "cursor"},
		PrimaryProvider:   domain.ExtratorProviderGemini,
		SecondaryProvider: domain.ExtratorProviderCursor,
		Cota:              cota,
		Location:          time.UTC,
		Clock:             func() time.Time { return time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC) },
	}
	_, _, _, err := f.Extract(context.Background(), []domain.PageImage{{Page: 1, JPEG: []byte{1}}})
	if !errors.Is(err, domain.ErrExtratorCota) {
		t.Fatalf("err=%v", err)
	}
}
