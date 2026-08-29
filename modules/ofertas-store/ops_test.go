package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store/storetest"
)

func TestOperacaoPipeline_EnqueueDequeueCancel(t *testing.T) {
	catalog := storetest.New(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 21, 8, 0, 0, 0, time.UTC)

	t.Run("enqueue_fica_pendente", func(t *testing.T) {
		if err := catalog.EnqueueOperacao(ctx, store.OperacaoPipeline{ID: "op1", Kind: store.OperacaoRun}); err != nil {
			t.Fatal(err)
		}
		got, ok, err := catalog.GetOperacao(ctx, "op1")
		if err != nil || !ok || got.Status != store.OperacaoPending {
			t.Fatalf("ok=%v err=%v got=%+v", ok, err, got)
		}
	})

	t.Run("dequeue_serial_marca_running", func(t *testing.T) {
		if err := catalog.EnqueueOperacao(ctx, store.OperacaoPipeline{ID: "op2", Kind: store.OperacaoDiscover, FonteID: "f1"}); err != nil {
			t.Fatal(err)
		}
		first, ok, err := catalog.DequeueOperacao(ctx, now)
		if err != nil || !ok || first.ID != "op1" || first.Status != store.OperacaoRunning {
			t.Fatalf("first ok=%v err=%v got=%+v", ok, err, first)
		}
		second, ok, err := catalog.DequeueOperacao(ctx, now.Add(time.Second))
		if err != nil || !ok || second.ID != "op2" || second.Status != store.OperacaoRunning {
			t.Fatalf("second ok=%v err=%v got=%+v", ok, err, second)
		}
		if _, ok, err := catalog.DequeueOperacao(ctx, now); err != nil || ok {
			t.Fatalf("empty queue ok=%v err=%v", ok, err)
		}
	})

	t.Run("cancel_so_pendente", func(t *testing.T) {
		if err := catalog.EnqueueOperacao(ctx, store.OperacaoPipeline{ID: "op3", Kind: store.OperacaoReprocess, DocumentoID: "d1"}); err != nil {
			t.Fatal(err)
		}
		got, err := catalog.CancelOperacao(ctx, "op3", now)
		if err != nil || got.Status != store.OperacaoCanceled {
			t.Fatalf("err=%v got=%+v", err, got)
		}
		if _, err := catalog.CancelOperacao(ctx, "op1", now); !errors.Is(err, store.ErrConflict) {
			t.Fatalf("cancel running err=%v want ErrConflict", err)
		}
	})
}
