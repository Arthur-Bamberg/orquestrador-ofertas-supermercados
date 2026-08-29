package pg

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, fmt.Errorf("DATABASE_URL obrigatório")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	s := &Store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS whatsapp`); err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS whatsapp.schema_migrations (version TEXT PRIMARY KEY)`); err != nil {
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
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM whatsapp.schema_migrations WHERE version=$1)`, name).Scan(&applied); err != nil {
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
		if _, err := tx.Exec(ctx, `INSERT INTO whatsapp.schema_migrations (version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) TruncateAll(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `TRUNCATE whatsapp.mensagem, whatsapp.conversa, whatsapp.contato RESTART IDENTITY CASCADE`)
	return err
}

func (s *Store) UpsertContato(ctx context.Context, c domain.Contato) (domain.Contato, error) {
	var existing domain.Contato
	err := s.pool.QueryRow(ctx, `SELECT id, jid FROM whatsapp.contato WHERE jid=$1`, string(c.JID)).Scan(&existing.ID, &existing.JID)
	if err == nil {
		return existing, nil
	}
	if err != pgx.ErrNoRows {
		return domain.Contato{}, err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO whatsapp.contato (id, jid) VALUES ($1, $2)`, string(c.ID), string(c.JID))
	if err != nil {
		return domain.Contato{}, err
	}
	return c, nil
}

func (s *Store) UpsertConversa(ctx context.Context, c domain.Conversa) (domain.Conversa, error) {
	var existing domain.Conversa
	var tipo string
	err := s.pool.QueryRow(ctx, `SELECT id, jid, tipo FROM whatsapp.conversa WHERE jid=$1`, string(c.JID)).Scan(&existing.ID, &existing.JID, &tipo)
	if err == nil {
		existing.Tipo = domain.TipoConversa(tipo)
		return existing, nil
	}
	if err != pgx.ErrNoRows {
		return domain.Conversa{}, err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO whatsapp.conversa (id, jid, tipo) VALUES ($1, $2, $3)`, string(c.ID), string(c.JID), string(c.Tipo))
	if err != nil {
		return domain.Conversa{}, err
	}
	return c, nil
}

func (s *Store) SalvarMensagem(ctx context.Context, m domain.Mensagem) error {
	midiaTipo, midiaPath, midiaFile, midiaMIME := "", "", "", ""
	if m.Midia != nil {
		midiaTipo = string(m.Midia.Tipo)
		midiaPath = m.Midia.Path
		midiaFile = m.Midia.Filename
		midiaMIME = m.Midia.MIME
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO whatsapp.mensagem (
			id, conversa_id, contato_id, direcao, corpo, provedor_id, status,
			midia_tipo, midia_path, midia_filename, midia_mime, criado_em
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			corpo = EXCLUDED.corpo,
			provedor_id = EXCLUDED.provedor_id,
			midia_tipo = EXCLUDED.midia_tipo,
			midia_path = EXCLUDED.midia_path,
			midia_filename = EXCLUDED.midia_filename,
			midia_mime = EXCLUDED.midia_mime
	`, string(m.ID), string(m.ConversaID), string(m.ContatoID), string(m.Direcao), m.Corpo, m.ProvedorID, string(m.Status),
		midiaTipo, midiaPath, midiaFile, midiaMIME, m.CriadoEm)
	return err
}

func (s *Store) MensagemPorProvedor(ctx context.Context, provedorID string) (domain.Mensagem, bool, error) {
	return s.scanOne(ctx, `SELECT `+msgCols+` FROM whatsapp.mensagem WHERE provedor_id=$1 AND provedor_id <> ''`, provedorID)
}

func (s *Store) GetMensagem(ctx context.Context, id domain.MensagemID) (domain.Mensagem, bool, error) {
	return s.scanOne(ctx, `SELECT `+msgCols+` FROM whatsapp.mensagem WHERE id=$1`, string(id))
}

func (s *Store) ListarMensagens(ctx context.Context, conversaID domain.ConversaID) ([]domain.Mensagem, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+msgCols+` FROM whatsapp.mensagem WHERE conversa_id=$1 ORDER BY criado_em, id`, string(conversaID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Mensagem
	for rows.Next() {
		m, err := scanMensagem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetConversa(ctx context.Context, id domain.ConversaID) (domain.Conversa, bool, error) {
	var c domain.Conversa
	var tipo string
	err := s.pool.QueryRow(ctx, `SELECT id, jid, tipo FROM whatsapp.conversa WHERE id=$1`, string(id)).Scan(&c.ID, &c.JID, &tipo)
	if err == pgx.ErrNoRows {
		return domain.Conversa{}, false, nil
	}
	if err != nil {
		return domain.Conversa{}, false, err
	}
	c.Tipo = domain.TipoConversa(tipo)
	return c, true, nil
}

func (s *Store) ConversaPorJID(ctx context.Context, jid domain.JID) (domain.Conversa, bool, error) {
	var c domain.Conversa
	var tipo string
	err := s.pool.QueryRow(ctx, `SELECT id, jid, tipo FROM whatsapp.conversa WHERE jid=$1`, string(jid)).Scan(&c.ID, &c.JID, &tipo)
	if err == pgx.ErrNoRows {
		return domain.Conversa{}, false, nil
	}
	if err != nil {
		return domain.Conversa{}, false, err
	}
	c.Tipo = domain.TipoConversa(tipo)
	return c, true, nil
}

const msgCols = `id, conversa_id, contato_id, direcao, corpo, provedor_id, status, midia_tipo, midia_path, midia_filename, midia_mime, criado_em`

type rowScanner interface {
	Scan(dest ...any) error
}

func (s *Store) scanOne(ctx context.Context, q string, arg any) (domain.Mensagem, bool, error) {
	m, err := scanMensagem(s.pool.QueryRow(ctx, q, arg))
	if err == pgx.ErrNoRows {
		return domain.Mensagem{}, false, nil
	}
	if err != nil {
		return domain.Mensagem{}, false, err
	}
	return m, true, nil
}

func scanMensagem(row rowScanner) (domain.Mensagem, error) {
	var m domain.Mensagem
	var midiaTipo, midiaPath, midiaFile, midiaMIME string
	err := row.Scan(&m.ID, &m.ConversaID, &m.ContatoID, &m.Direcao, &m.Corpo, &m.ProvedorID, &m.Status,
		&midiaTipo, &midiaPath, &midiaFile, &midiaMIME, &m.CriadoEm)
	if err != nil {
		return domain.Mensagem{}, err
	}
	if midiaTipo != "" || midiaPath != "" {
		m.Midia = &domain.Midia{Tipo: domain.TipoMidia(midiaTipo), Path: midiaPath, Filename: midiaFile, MIME: midiaMIME}
	}
	if m.CriadoEm.IsZero() {
		m.CriadoEm = time.Time{}
	}
	return m, nil
}
