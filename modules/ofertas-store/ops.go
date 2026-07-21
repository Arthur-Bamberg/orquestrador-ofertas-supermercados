package store

import (
	"context"
	"fmt"
	"sort"
	"time"
)

const (
	opsQueueKey   = "ofertas-api:ops:queue"
	opsHistoryKey = "ofertas-api:ops:history"
)

func opsKey(id string) string { return "ofertas-api:ops:" + id }

type OperacaoKind string

const (
	OperacaoRun       OperacaoKind = "run"
	OperacaoDiscover  OperacaoKind = "discover"
	OperacaoReprocess OperacaoKind = "reprocess"
)

type OperacaoStatus string

const (
	OperacaoPending   OperacaoStatus = "pending"
	OperacaoRunning   OperacaoStatus = "running"
	OperacaoSucceeded OperacaoStatus = "succeeded"
	OperacaoFailed    OperacaoStatus = "failed"
	OperacaoCanceled  OperacaoStatus = "canceled"
)

type OperacaoPipeline struct {
	ID          string         `json:"id"`
	Kind        OperacaoKind   `json:"kind"`
	Status      OperacaoStatus `json:"status"`
	FonteID     FonteID        `json:"fonteId,omitempty"`
	DocumentoID DocumentoID    `json:"documentoId,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	StartedAt   *time.Time     `json:"startedAt,omitempty"`
	FinishedAt  *time.Time     `json:"finishedAt,omitempty"`
	LogSnippet  string         `json:"logSnippet,omitempty"`
	Error       string         `json:"error,omitempty"`
}

func (s *Catalog) EnqueueOperacao(ctx context.Context, op OperacaoPipeline) error {
	if op.ID == "" {
		return fmt.Errorf("%w: operação id obrigatório", ErrInvalid)
	}
	if op.Kind == "" {
		return fmt.Errorf("%w: operação kind obrigatório", ErrInvalid)
	}
	if op.CreatedAt.IsZero() {
		op.CreatedAt = time.Now().UTC()
	}
	op.Status = OperacaoPending
	if err := s.saveJSON(ctx, opsKey(op.ID), op); err != nil {
		return err
	}
	queue, err := s.loadStringList(ctx, opsQueueKey)
	if err != nil {
		return err
	}
	queue = append(queue, op.ID)
	if err := s.saveJSON(ctx, opsQueueKey, queue); err != nil {
		return err
	}
	history, err := s.loadStringList(ctx, opsHistoryKey)
	if err != nil {
		return err
	}
	history = append(history, op.ID)
	return s.saveJSON(ctx, opsHistoryKey, history)
}

func (s *Catalog) GetOperacao(ctx context.Context, id string) (OperacaoPipeline, bool, error) {
	var op OperacaoPipeline
	ok, err := s.getJSON(ctx, opsKey(id), &op)
	return op, ok, err
}

func (s *Catalog) SaveOperacao(ctx context.Context, op OperacaoPipeline) error {
	if op.ID == "" {
		return fmt.Errorf("%w: operação id obrigatório", ErrInvalid)
	}
	return s.saveJSON(ctx, opsKey(op.ID), op)
}

func (s *Catalog) ListOperacoes(ctx context.Context) ([]OperacaoPipeline, error) {
	ids, err := s.loadStringList(ctx, opsHistoryKey)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		keys, err := s.c.Keys(ctx, "ofertas-api:ops:*")
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			if key == opsQueueKey || key == opsHistoryKey {
				continue
			}
			ids = append(ids, key[len("ofertas-api:ops:"):])
		}
	}
	out := make([]OperacaoPipeline, 0, len(ids))
	for _, id := range ids {
		op, ok, err := s.GetOperacao(ctx, id)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, op)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (s *Catalog) DequeueOperacao(ctx context.Context, now time.Time) (OperacaoPipeline, bool, error) {
	queue, err := s.loadStringList(ctx, opsQueueKey)
	if err != nil {
		return OperacaoPipeline{}, false, err
	}
	nextQueue := make([]string, 0, len(queue))
	for i, id := range queue {
		op, ok, err := s.GetOperacao(ctx, id)
		if err != nil {
			return OperacaoPipeline{}, false, err
		}
		if !ok || op.Status != OperacaoPending {
			continue
		}
		started := now.UTC()
		op.Status = OperacaoRunning
		op.StartedAt = &started
		if err := s.SaveOperacao(ctx, op); err != nil {
			return OperacaoPipeline{}, false, err
		}
		nextQueue = append(nextQueue, queue[:i]...)
		nextQueue = append(nextQueue, queue[i+1:]...)
		if err := s.saveJSON(ctx, opsQueueKey, nextQueue); err != nil {
			return OperacaoPipeline{}, false, err
		}
		return op, true, nil
	}
	return OperacaoPipeline{}, false, nil
}

func (s *Catalog) CancelOperacao(ctx context.Context, id string, now time.Time) (OperacaoPipeline, error) {
	op, ok, err := s.GetOperacao(ctx, id)
	if err != nil {
		return OperacaoPipeline{}, err
	}
	if !ok {
		return OperacaoPipeline{}, fmt.Errorf("%w: operação %s", ErrNotFound, id)
	}
	if op.Status != OperacaoPending {
		return OperacaoPipeline{}, fmt.Errorf("%w: apenas operações pendentes podem ser canceladas", ErrConflict)
	}
	finished := now.UTC()
	op.Status = OperacaoCanceled
	op.FinishedAt = &finished
	if err := s.SaveOperacao(ctx, op); err != nil {
		return OperacaoPipeline{}, err
	}
	queue, err := s.loadStringList(ctx, opsQueueKey)
	if err != nil {
		return OperacaoPipeline{}, err
	}
	next := queue[:0]
	for _, queuedID := range queue {
		if queuedID != id {
			next = append(next, queuedID)
		}
	}
	return op, s.saveJSON(ctx, opsQueueKey, next)
}

func (s *Catalog) loadStringList(ctx context.Context, key string) ([]string, error) {
	var out []string
	ok, err := s.getJSON(ctx, key, &out)
	if err != nil || !ok {
		return nil, err
	}
	return out, nil
}
