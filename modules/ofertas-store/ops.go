package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

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

const operacaoSelect = `SELECT id, kind, status, fonte_id, documento_id, created_at, started_at, finished_at, log_snippet, error FROM operacao_pipeline`

func scanOperacao(row rowScanner) (OperacaoPipeline, error) {
	var op OperacaoPipeline
	var kind, status, fonte, doc string
	err := row.Scan(&op.ID, &kind, &status, &fonte, &doc, &op.CreatedAt, &op.StartedAt, &op.FinishedAt, &op.LogSnippet, &op.Error)
	op.Kind = OperacaoKind(kind)
	op.Status = OperacaoStatus(status)
	op.FonteID = FonteID(fonte)
	op.DocumentoID = DocumentoID(doc)
	return op, err
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
	_, err := s.pool.Exec(ctx, `INSERT INTO operacao_pipeline (
			id, kind, status, fonte_id, documento_id, created_at, log_snippet, error, queue_pos)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8, nextval('operacao_pipeline_queue_seq'))`,
		op.ID, string(op.Kind), string(op.Status), op.FonteID, op.DocumentoID, op.CreatedAt, op.LogSnippet, op.Error)
	return wrapPG(err)
}

func (s *Catalog) UpsertOperacao(ctx context.Context, op OperacaoPipeline) error {
	if op.ID == "" {
		return fmt.Errorf("%w: operação id obrigatório", ErrInvalid)
	}
	if op.Kind == "" {
		return fmt.Errorf("%w: operação kind obrigatório", ErrInvalid)
	}
	if op.CreatedAt.IsZero() {
		op.CreatedAt = time.Now().UTC()
	}
	if op.Status == "" {
		op.Status = OperacaoPending
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO operacao_pipeline (
			id, kind, status, fonte_id, documento_id, created_at, started_at, finished_at, log_snippet, error, queue_pos)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, nextval('operacao_pipeline_queue_seq'))
		ON CONFLICT (id) DO UPDATE SET
			kind = EXCLUDED.kind, status = EXCLUDED.status, fonte_id = EXCLUDED.fonte_id,
			documento_id = EXCLUDED.documento_id, created_at = EXCLUDED.created_at,
			started_at = EXCLUDED.started_at, finished_at = EXCLUDED.finished_at,
			log_snippet = EXCLUDED.log_snippet, error = EXCLUDED.error`,
		op.ID, string(op.Kind), string(op.Status), op.FonteID, op.DocumentoID, op.CreatedAt,
		op.StartedAt, op.FinishedAt, op.LogSnippet, op.Error)
	return wrapPG(err)
}

func (s *Catalog) GetOperacao(ctx context.Context, id string) (OperacaoPipeline, bool, error) {
	op, err := scanOperacao(s.pool.QueryRow(ctx, operacaoSelect+" WHERE id = $1", id))
	if errorsIsNoRows(err) {
		return OperacaoPipeline{}, false, nil
	}
	return op, err == nil, wrapPG(err)
}

func errorsIsNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func (s *Catalog) SaveOperacao(ctx context.Context, op OperacaoPipeline) error {
	if op.ID == "" {
		return fmt.Errorf("%w: operação id obrigatório", ErrInvalid)
	}
	_, err := s.pool.Exec(ctx, `UPDATE operacao_pipeline SET
			kind=$2, status=$3, fonte_id=$4, documento_id=$5, created_at=$6,
			started_at=$7, finished_at=$8, log_snippet=$9, error=$10
		WHERE id=$1`,
		op.ID, string(op.Kind), string(op.Status), op.FonteID, op.DocumentoID, op.CreatedAt,
		op.StartedAt, op.FinishedAt, op.LogSnippet, op.Error)
	return wrapPG(err)
}

func (s *Catalog) ListOperacoes(ctx context.Context) ([]OperacaoPipeline, error) {
	rows, err := s.pool.Query(ctx, operacaoSelect+" ORDER BY created_at DESC, id")
	if err != nil {
		return nil, wrapPG(err)
	}
	defer rows.Close()
	out := []OperacaoPipeline{}
	for rows.Next() {
		op, err := scanOperacao(rows)
		if err != nil {
			return nil, wrapPG(err)
		}
		out = append(out, op)
	}
	return out, wrapPG(rows.Err())
}

func (s *Catalog) DequeueOperacao(ctx context.Context, now time.Time) (OperacaoPipeline, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return OperacaoPipeline{}, false, wrapPG(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	op, err := scanOperacao(tx.QueryRow(ctx, operacaoSelect+`
		WHERE id = (
			SELECT id FROM operacao_pipeline
			WHERE status = $1
			ORDER BY queue_pos NULLS LAST, created_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)`, string(OperacaoPending)))
	if errorsIsNoRows(err) {
		return OperacaoPipeline{}, false, wrapPG(tx.Commit(ctx))
	}
	if err != nil {
		return OperacaoPipeline{}, false, wrapPG(err)
	}
	started := now.UTC()
	op.Status = OperacaoRunning
	op.StartedAt = &started
	if _, err := tx.Exec(ctx, `UPDATE operacao_pipeline SET status=$2, started_at=$3 WHERE id=$1`,
		op.ID, string(op.Status), op.StartedAt); err != nil {
		return OperacaoPipeline{}, false, wrapPG(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return OperacaoPipeline{}, false, wrapPG(err)
	}
	return op, true, nil
}

func (s *Catalog) CancelOperacao(ctx context.Context, id string, now time.Time) (OperacaoPipeline, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return OperacaoPipeline{}, wrapPG(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	op, err := scanOperacao(tx.QueryRow(ctx, operacaoSelect+" WHERE id = $1 FOR UPDATE", id))
	if errorsIsNoRows(err) {
		return OperacaoPipeline{}, fmt.Errorf("%w: operação %s", ErrNotFound, id)
	}
	if err != nil {
		return OperacaoPipeline{}, wrapPG(err)
	}
	if op.Status != OperacaoPending {
		return OperacaoPipeline{}, fmt.Errorf("%w: apenas operações pendentes podem ser canceladas", ErrConflict)
	}
	finished := now.UTC()
	op.Status = OperacaoCanceled
	op.FinishedAt = &finished
	if _, err := tx.Exec(ctx, `UPDATE operacao_pipeline SET status=$2, finished_at=$3 WHERE id=$1`,
		op.ID, string(op.Status), op.FinishedAt); err != nil {
		return OperacaoPipeline{}, wrapPG(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return OperacaoPipeline{}, wrapPG(err)
	}
	return op, nil
}
