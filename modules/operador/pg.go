package operador

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type PG struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*PG, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, fmt.Errorf("DATABASE_URL obrigatório")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	s := &PG{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *PG) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func (s *PG) migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `SELECT pg_advisory_lock(8810018)`); err != nil {
		return err
	}
	defer func() { _, _ = s.pool.Exec(ctx, `SELECT pg_advisory_unlock(8810018)`) }()

	if _, err := s.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS backoffice_schema_migrations (version TEXT PRIMARY KEY)`); err != nil {
		return err
	}
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		var applied bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM backoffice_schema_migrations WHERE version=$1)`, name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrate %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO backoffice_schema_migrations (version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *PG) Truncate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `TRUNCATE backoffice.identificacao, backoffice.operador`)
	return err
}

func (s *PG) Criar(ctx context.Context, nome, senha string) (Operador, error) {
	nome = normalizaNome(nome)
	if nome == "" {
		return Operador{}, ErrNome
	}
	hash, err := HashSenha(senha)
	if err != nil {
		return Operador{}, err
	}
	op := Operador{ID: newID(), Nome: nome}
	_, err = s.pool.Exec(ctx, `INSERT INTO backoffice.operador (id, nome, senha_hash) VALUES ($1, $2, $3)`, op.ID, op.Nome, hash)
	if err != nil {
		return Operador{}, err
	}
	return op, nil
}

func (s *PG) Identificar(ctx context.Context, nome, senha string) (Identificacao, error) {
	nome = normalizaNome(nome)
	var op Operador
	var hash string
	err := s.pool.QueryRow(ctx, `SELECT id, nome, senha_hash FROM backoffice.operador WHERE nome=$1`, nome).Scan(&op.ID, &op.Nome, &hash)
	if err != nil {
		if errorsIsNoRows(err) {
			return Identificacao{}, ErrCredencial
		}
		return Identificacao{}, err
	}
	if !SenhaConfere(hash, senha) {
		return Identificacao{}, ErrCredencial
	}
	id := Identificacao{ID: newID(), Operador: op}
	_, err = s.pool.Exec(ctx, `INSERT INTO backoffice.identificacao (id, operador_id) VALUES ($1, $2)`, id.ID, op.ID)
	if err != nil {
		return Identificacao{}, err
	}
	return id, nil
}

func (s *PG) OperadorPorIdentificacao(ctx context.Context, identificacaoID string) (Operador, bool, error) {
	if identificacaoID == "" {
		return Operador{}, false, nil
	}
	var op Operador
	err := s.pool.QueryRow(ctx, `
		SELECT o.id, o.nome
		FROM backoffice.identificacao i
		JOIN backoffice.operador o ON o.id = i.operador_id
		WHERE i.id=$1`, identificacaoID).Scan(&op.ID, &op.Nome)
	if err != nil {
		if errorsIsNoRows(err) {
			return Operador{}, false, nil
		}
		return Operador{}, false, err
	}
	return op, true, nil
}

func (s *PG) Sair(ctx context.Context, identificacaoID string) error {
	if identificacaoID == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM backoffice.identificacao WHERE id=$1`, identificacaoID)
	return err
}

func (s *PG) Apagar(ctx context.Context, operadorID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM backoffice.operador WHERE id=$1`, operadorID)
	return err
}

func (s *PG) DefinirSenha(ctx context.Context, operadorID, senha string) error {
	hash, err := HashSenha(senha)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `UPDATE backoffice.operador SET senha_hash=$1 WHERE id=$2`, hash, operadorID)
	return err
}

func errorsIsNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
