package operador_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store/storetest"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/operador"
)

func TestMain(m *testing.M) {
	code := m.Run()
	storetest.Stop()
	os.Exit(code)
}

func TestIdentificar_nomeESenhaGeramIdentificacao(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	op, err := s.Criar(ctx, "arthur", "segredo")
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.Identificar(ctx, "arthur", "segredo")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == "" || got.Operador.ID != op.ID || got.Operador.Nome != "arthur" {
		t.Fatalf("%+v", got)
	}

	lido, ok, err := s.OperadorPorIdentificacao(ctx, got.ID)
	if err != nil || !ok || lido.Nome != "arthur" {
		t.Fatalf("consulta %+v ok=%v err=%v", lido, ok, err)
	}
}

func TestIdentificar_nomeOuSenhaInvalidosSaoOMesmoErro(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	if _, err := s.Criar(ctx, "arthur", "segredo"); err != nil {
		t.Fatal(err)
	}

	_, err := s.Identificar(ctx, "arthur", "errada")
	if !errors.Is(err, operador.ErrCredencial) {
		t.Fatalf("senha: %v", err)
	}
	_, err = s.Identificar(ctx, "maria", "segredo")
	if !errors.Is(err, operador.ErrCredencial) {
		t.Fatalf("nome: %v", err)
	}
}

func TestSair_soAquelaIdentificacao(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	if _, err := s.Criar(ctx, "arthur", "segredo"); err != nil {
		t.Fatal(err)
	}
	a, err := s.Identificar(ctx, "arthur", "segredo")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Identificar(ctx, "arthur", "segredo")
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Sair(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.OperadorPorIdentificacao(ctx, a.ID); ok {
		t.Fatal("A deveria ter saído")
	}
	if _, ok, _ := s.OperadorPorIdentificacao(ctx, b.ID); !ok {
		t.Fatal("B deveria permanecer")
	}
}

func TestApagarOperador_invalidaTodasIdentificacoes(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	op, err := s.Criar(ctx, "maria", "segredo")
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.Identificar(ctx, "maria", "segredo")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Identificar(ctx, "maria", "segredo")
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Apagar(ctx, op.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.OperadorPorIdentificacao(ctx, a.ID); ok {
		t.Fatal("A")
	}
	if _, ok, _ := s.OperadorPorIdentificacao(ctx, b.ID); ok {
		t.Fatal("B")
	}
}

func TestDefinirSenha_naoInvalidaIdentificacao(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	op, err := s.Criar(ctx, "arthur", "antiga")
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.Identificar(ctx, "arthur", "antiga")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DefinirSenha(ctx, op.ID, "nova"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.OperadorPorIdentificacao(ctx, id.ID); err != nil || !ok {
		t.Fatalf("ainda identificado: ok=%v err=%v", ok, err)
	}
	if _, err := s.Identificar(ctx, "arthur", "antiga"); !errors.Is(err, operador.ErrCredencial) {
		t.Fatalf("próximo identificar usa senha nova: %v", err)
	}
}

func openStore(t *testing.T) *operador.PG {
	t.Helper()
	_ = storetest.New(t)
	ctx := context.Background()
	s, err := operador.Open(ctx, storetest.DSN(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	if err := s.Truncate(ctx); err != nil {
		t.Fatal(err)
	}
	return s
}
