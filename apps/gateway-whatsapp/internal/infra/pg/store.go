package pg

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	existing, ok, err := s.findContato(ctx, c.JID, c.JIDLID)
	if err != nil {
		return domain.Contato{}, err
	}
	if ok {
		return s.mergeContato(ctx, existing, c)
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO whatsapp.contato (id, jid, jid_lid) VALUES ($1, $2, $3)`,
		string(c.ID), string(c.JID), string(c.JIDLID))
	if err == nil {
		return c, nil
	}
	if !isUniqueViolation(err) {
		return domain.Contato{}, err
	}
	existing, ok, findErr := s.findContato(ctx, c.JID, c.JIDLID)
	if findErr != nil {
		return domain.Contato{}, findErr
	}
	if !ok {
		return domain.Contato{}, err
	}
	return s.mergeContato(ctx, existing, c)
}

func (s *Store) findContato(ctx context.Context, keys ...domain.JID) (domain.Contato, bool, error) {
	var c domain.Contato
	var lid string
	for _, k := range keys {
		if k == "" {
			continue
		}
		err := s.pool.QueryRow(ctx, `SELECT id, jid, jid_lid FROM whatsapp.contato WHERE jid=$1 OR jid_lid=$1`, string(k)).Scan(&c.ID, &c.JID, &lid)
		if err == pgx.ErrNoRows {
			continue
		}
		if err != nil {
			return domain.Contato{}, false, err
		}
		c.JIDLID = domain.JID(lid)
		return c, true, nil
	}
	return domain.Contato{}, false, nil
}

func (s *Store) mergeContato(ctx context.Context, existing, in domain.Contato) (domain.Contato, error) {
	jid, lid := existing.JID, existing.JIDLID
	if lid == "" && in.JIDLID != "" {
		lid = in.JIDLID
	}
	if strings.HasSuffix(string(jid), "@lid") && in.JID != "" && !strings.HasSuffix(string(in.JID), "@lid") {
		jid = in.JID
	}
	if jid == existing.JID && lid == existing.JIDLID {
		return existing, nil
	}
	_, err := s.pool.Exec(ctx, `UPDATE whatsapp.contato SET jid=$2, jid_lid=$3 WHERE id=$1`, string(existing.ID), string(jid), string(lid))
	if err != nil {
		return domain.Contato{}, err
	}
	existing.JID, existing.JIDLID = jid, lid
	return existing, nil
}

func (s *Store) UpsertConversa(ctx context.Context, c domain.Conversa) (domain.Conversa, error) {
	existing, ok, err := s.findConversa(ctx, c.JID, c.JIDLID)
	if err != nil {
		return domain.Conversa{}, err
	}
	if ok {
		return s.mergeConversa(ctx, existing, c)
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO whatsapp.conversa (id, jid, jid_lid, tipo) VALUES ($1, $2, $3, $4)`,
		string(c.ID), string(c.JID), string(c.JIDLID), string(c.Tipo))
	if err == nil {
		return c, nil
	}
	if !isUniqueViolation(err) {
		return domain.Conversa{}, err
	}
	existing, ok, findErr := s.findConversa(ctx, c.JID, c.JIDLID)
	if findErr != nil {
		return domain.Conversa{}, findErr
	}
	if !ok {
		return domain.Conversa{}, err
	}
	return s.mergeConversa(ctx, existing, c)
}

func (s *Store) findConversa(ctx context.Context, keys ...domain.JID) (domain.Conversa, bool, error) {
	var c domain.Conversa
	var tipo, lid string
	for _, k := range keys {
		if k == "" {
			continue
		}
		err := s.pool.QueryRow(ctx, `SELECT id, jid, jid_lid, tipo FROM whatsapp.conversa WHERE jid=$1 OR jid_lid=$1`, string(k)).Scan(&c.ID, &c.JID, &lid, &tipo)
		if err == pgx.ErrNoRows {
			continue
		}
		if err != nil {
			return domain.Conversa{}, false, err
		}
		c.JIDLID = domain.JID(lid)
		c.Tipo = domain.TipoConversa(tipo)
		return c, true, nil
	}
	return domain.Conversa{}, false, nil
}

func (s *Store) mergeConversa(ctx context.Context, existing, in domain.Conversa) (domain.Conversa, error) {
	jid, lid := existing.JID, existing.JIDLID
	if lid == "" && in.JIDLID != "" {
		lid = in.JIDLID
	}
	if strings.HasSuffix(string(jid), "@lid") && in.JID != "" && !strings.HasSuffix(string(in.JID), "@lid") {
		jid = in.JID
	}
	if jid == existing.JID && lid == existing.JIDLID {
		return existing, nil
	}
	_, err := s.pool.Exec(ctx, `UPDATE whatsapp.conversa SET jid=$2, jid_lid=$3 WHERE id=$1`, string(existing.ID), string(jid), string(lid))
	if err != nil {
		return domain.Conversa{}, err
	}
	existing.JID, existing.JIDLID = jid, lid
	return existing, nil
}

func (s *Store) SalvarMensagem(ctx context.Context, m domain.Mensagem) error {
	midiaTipo, midiaPath, midiaFile, midiaMIME := "", "", "", ""
	if m.Midia != nil {
		midiaTipo = string(m.Midia.Tipo)
		midiaPath = m.Midia.Path
		midiaFile = m.Midia.Filename
		midiaMIME = m.Midia.MIME
	}
	origem := string(m.Origem)
	if origem == "" {
		origem = string(domain.OrigemVivo)
	}
	payload := payloadJSON(m.Payload)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO whatsapp.mensagem (
			id, conversa_id, contato_id, direcao, corpo, provedor_id, status,
			midia_tipo, midia_path, midia_filename, midia_mime, criado_em,
			origem, push_name, tipo, payload
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16::jsonb)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			corpo = EXCLUDED.corpo,
			provedor_id = EXCLUDED.provedor_id,
			midia_tipo = EXCLUDED.midia_tipo,
			midia_path = EXCLUDED.midia_path,
			midia_filename = EXCLUDED.midia_filename,
			midia_mime = EXCLUDED.midia_mime,
			origem = EXCLUDED.origem,
			push_name = EXCLUDED.push_name,
			tipo = EXCLUDED.tipo,
			payload = EXCLUDED.payload
	`, string(m.ID), string(m.ConversaID), string(m.ContatoID), string(m.Direcao), m.Corpo, m.ProvedorID, string(m.Status),
		midiaTipo, midiaPath, midiaFile, midiaMIME, m.CriadoEm,
		origem, m.PushName, string(m.Tipo), payload)
	if err != nil && isUniqueViolation(err) && m.ProvedorID != "" {
		return domain.ErrProvedorDuplicado
	}
	return err
}

func isUniqueViolation(err error) bool {
	var e *pgconn.PgError
	return errors.As(err, &e) && e.Code == "23505"
}

func payloadJSON(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "{}"
	}
	return raw
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

func (s *Store) ListarConversas(ctx context.Context) ([]domain.ConversaResumo, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.jid, c.jid_lid, c.tipo,
			(SELECT count(*) FROM whatsapp.mensagem m WHERE m.conversa_id = c.id),
			`+msgColsPrefixed("u")+`
		FROM whatsapp.conversa c
		LEFT JOIN LATERAL (
			SELECT `+msgCols+`
			FROM whatsapp.mensagem
			WHERE conversa_id = c.id
			ORDER BY criado_em DESC, id DESC
			LIMIT 1
		) u ON true
		ORDER BY u.criado_em DESC NULLS LAST, c.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ConversaResumo
	for rows.Next() {
		var r domain.ConversaResumo
		var tipo, lid string
		var total int
		var midiaTipo, midiaPath, midiaFile, midiaMIME, origem, msgTipo *string
		var payload []byte
		var msgID, convID, contatoID, direcao, corpo, provedorID, status *string
		var criadoEm *time.Time
		var pushName *string
		err := rows.Scan(
			&r.ID, &r.JID, &lid, &tipo, &total,
			&msgID, &convID, &contatoID, &direcao, &corpo, &provedorID, &status,
			&midiaTipo, &midiaPath, &midiaFile, &midiaMIME, &criadoEm, &origem, &pushName, &msgTipo, &payload,
		)
		if err != nil {
			return nil, err
		}
		r.JIDLID = domain.JID(lid)
		r.Tipo = domain.TipoConversa(tipo)
		r.TotalMensagens = total
		if msgID != nil {
			m := domain.Mensagem{
				ID:         domain.MensagemID(*msgID),
				ConversaID: domain.ConversaID(deref(convID)),
				ContatoID:  domain.ContatoID(deref(contatoID)),
				Direcao:    domain.Direcao(deref(direcao)),
				Corpo:      deref(corpo),
				ProvedorID: deref(provedorID),
				Status:     domain.StatusEnvio(deref(status)),
				Origem:     domain.OrigemMensagem(deref(origem)),
				PushName:   deref(pushName),
				Tipo:       domain.TipoMensagem(deref(msgTipo)),
			}
			if criadoEm != nil {
				m.CriadoEm = *criadoEm
			}
			if deref(midiaTipo) != "" || deref(midiaPath) != "" {
				m.Midia = &domain.Midia{Tipo: domain.TipoMidia(deref(midiaTipo)), Path: deref(midiaPath), Filename: deref(midiaFile), MIME: deref(midiaMIME)}
			}
			m.Payload = strings.TrimSpace(string(payload))
			if m.Payload == "{}" {
				m.Payload = ""
			}
			r.UltimaMensagem = &m
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func msgColsPrefixed(alias string) string {
	parts := strings.Split(msgCols, ", ")
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = alias + "." + p
	}
	return strings.Join(out, ", ")
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (s *Store) GetConversa(ctx context.Context, id domain.ConversaID) (domain.Conversa, bool, error) {
	var c domain.Conversa
	var tipo, lid string
	err := s.pool.QueryRow(ctx, `SELECT id, jid, jid_lid, tipo FROM whatsapp.conversa WHERE id=$1`, string(id)).Scan(&c.ID, &c.JID, &lid, &tipo)
	if err == pgx.ErrNoRows {
		return domain.Conversa{}, false, nil
	}
	if err != nil {
		return domain.Conversa{}, false, err
	}
	c.JIDLID = domain.JID(lid)
	c.Tipo = domain.TipoConversa(tipo)
	return c, true, nil
}

func (s *Store) ConversaPorJID(ctx context.Context, jid domain.JID) (domain.Conversa, bool, error) {
	return s.findConversa(ctx, jid)
}

func (s *Store) MarcarRecibo(ctx context.Context, provedorID string, status domain.StatusEnvio) (bool, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE whatsapp.mensagem SET status=$2 WHERE provedor_id=$1 AND provedor_id <> ''`, provedorID, string(status))
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

const msgCols = `id, conversa_id, contato_id, direcao, corpo, provedor_id, status, midia_tipo, midia_path, midia_filename, midia_mime, criado_em, origem, push_name, tipo, payload`

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
	var midiaTipo, midiaPath, midiaFile, midiaMIME, origem, tipo string
	var payload []byte
	err := row.Scan(&m.ID, &m.ConversaID, &m.ContatoID, &m.Direcao, &m.Corpo, &m.ProvedorID, &m.Status,
		&midiaTipo, &midiaPath, &midiaFile, &midiaMIME, &m.CriadoEm, &origem, &m.PushName, &tipo, &payload)
	if err != nil {
		return domain.Mensagem{}, err
	}
	if midiaTipo != "" || midiaPath != "" {
		m.Midia = &domain.Midia{Tipo: domain.TipoMidia(midiaTipo), Path: midiaPath, Filename: midiaFile, MIME: midiaMIME}
	}
	m.Origem = domain.OrigemMensagem(origem)
	m.Tipo = domain.TipoMensagem(tipo)
	m.Payload = strings.TrimSpace(string(payload))
	if m.Payload == "{}" {
		m.Payload = ""
	}
	return m, nil
}
