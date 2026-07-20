package extrator

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

// Failover tries primary Extrator then secondary on rate-limit (ADR 0037).
type Failover struct {
	Primary         domain.Extrator
	Secondary       domain.Extrator
	PrimaryProvider string
	SecondaryProvider string
	Cota            domain.ExtratorCotaStore
	Location        *time.Location
	Log             *log.Logger
	Clock           func() time.Time
}

func (f *Failover) dia() string {
	clock := f.Clock
	if clock == nil {
		clock = time.Now
	}
	loc := f.Location
	if loc == nil {
		loc = time.UTC
	}
	return clock().In(loc).Format("2006-01-02")
}

func (f *Failover) logf(format string, args ...any) {
	if f.Log != nil {
		f.Log.Printf(format, args...)
	}
}

func (f *Failover) Extract(ctx context.Context, images []domain.PageImage) ([]domain.CandidatoOferta, []byte, *domain.UsoExtrator, error) {
	dia := f.dia()
	primaryOK := f.Primary != nil
	secondaryOK := f.Secondary != nil

	if primaryOK && f.Cota != nil {
		esgotado, err := f.Cota.Esgotado(ctx, f.PrimaryProvider, dia)
		if err != nil {
			return nil, nil, nil, err
		}
		if esgotado {
			f.logf("extrator %s esgotado no dia %s; pulando", f.PrimaryProvider, dia)
			primaryOK = false
		}
	}
	if secondaryOK && f.Cota != nil {
		esgotado, err := f.Cota.Esgotado(ctx, f.SecondaryProvider, dia)
		if err != nil {
			return nil, nil, nil, err
		}
		if esgotado {
			f.logf("extrator %s esgotado no dia %s; pulando", f.SecondaryProvider, dia)
			secondaryOK = false
		}
	}
	if !primaryOK && !secondaryOK {
		return nil, nil, nil, fmt.Errorf("%w: nenhum adapter disponível no dia %s", domain.ErrExtratorCota, dia)
	}

	if primaryOK {
		cands, raw, uso, err := f.Primary.Extract(ctx, images)
		if err == nil {
			if uso != nil && uso.Provider == "" {
				uso.Provider = f.PrimaryProvider
			}
			return cands, raw, uso, nil
		}
		if errors.Is(err, domain.ErrExtratorCota) {
			if f.Cota != nil {
				if markErr := f.Cota.MarcarEsgotado(ctx, f.PrimaryProvider, dia); markErr != nil {
					return nil, nil, nil, markErr
				}
			}
			f.logf("extrator %s cota esgotada; failover", f.PrimaryProvider)
			primaryOK = false
		} else {
			return nil, nil, nil, err
		}
	}

	if !secondaryOK {
		return nil, nil, nil, fmt.Errorf("%w: %s esgotado e sem secondary", domain.ErrExtratorCota, f.PrimaryProvider)
	}

	cands, raw, uso, err := f.Secondary.Extract(ctx, images)
	if err == nil {
		if uso != nil && uso.Provider == "" {
			uso.Provider = f.SecondaryProvider
		}
		return cands, raw, uso, nil
	}
	if errors.Is(err, domain.ErrExtratorCota) {
		if f.Cota != nil {
			if markErr := f.Cota.MarcarEsgotado(ctx, f.SecondaryProvider, dia); markErr != nil {
				return nil, nil, nil, markErr
			}
		}
		return nil, nil, nil, fmt.Errorf("%w: %s", domain.ErrExtratorCota, f.SecondaryProvider)
	}
	return nil, nil, nil, err
}
