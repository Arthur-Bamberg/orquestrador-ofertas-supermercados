package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrDeleteBlocked = errors.New("delete blocked")
	ErrConflict      = errors.New("conflict")
	ErrInvalid       = errors.New("invalid")
)

func wrapPG(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23001":
			return fmt.Errorf("%w: %s", ErrDeleteBlocked, pgErr.Message)
		case "23503":
			if strings.Contains(pgErr.Message, "update or delete") {
				return fmt.Errorf("%w: %s", ErrDeleteBlocked, pgErr.Message)
			}
			return fmt.Errorf("%w: %s", ErrInvalid, pgErr.Message)
		case "23505":
			return fmt.Errorf("%w: %s", ErrConflict, pgErr.Message)
		}
	}
	return err
}

func (s *Catalog) SaveMercado(ctx context.Context, m Mercado) error {
	if m.ID == "" {
		return fmt.Errorf("%w: mercado id obrigatório", ErrInvalid)
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO mercado (id, nome) VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET nome = EXCLUDED.nome`, m.ID, m.Nome)
	return wrapPG(err)
}

func (s *Catalog) GetMercado(ctx context.Context, id MercadoID) (Mercado, bool, error) {
	var m Mercado
	err := s.pool.QueryRow(ctx, `SELECT id, nome FROM mercado WHERE id = $1`, id).Scan(&m.ID, &m.Nome)
	if errors.Is(err, pgx.ErrNoRows) {
		return Mercado{}, false, nil
	}
	return m, err == nil, wrapPG(err)
}

func (s *Catalog) ListMercados(ctx context.Context) ([]Mercado, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, nome FROM mercado ORDER BY id`)
	if err != nil {
		return nil, wrapPG(err)
	}
	defer rows.Close()
	var out []Mercado
	for rows.Next() {
		var m Mercado
		if err := rows.Scan(&m.ID, &m.Nome); err != nil {
			return nil, wrapPG(err)
		}
		out = append(out, m)
	}
	if out == nil {
		out = []Mercado{}
	}
	return out, wrapPG(rows.Err())
}

func (s *Catalog) DeleteMercado(ctx context.Context, id MercadoID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM mercado WHERE id = $1`, id)
	if err != nil {
		return wrapPG(err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: mercado %s", ErrNotFound, id)
	}
	return nil
}

func (s *Catalog) SaveFonte(ctx context.Context, f Fonte) error {
	if f.ID == "" {
		return fmt.Errorf("%w: fonte id obrigatório", ErrInvalid)
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO fonte (id, mercado_id, url, filtro_nome_documento, ativa)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET mercado_id = EXCLUDED.mercado_id, url = EXCLUDED.url,
			filtro_nome_documento = EXCLUDED.filtro_nome_documento, ativa = EXCLUDED.ativa`,
		f.ID, f.MercadoID, f.URL, f.FiltroNomeDocumento, f.IsAtiva())
	return wrapPG(err)
}

func (s *Catalog) GetFonte(ctx context.Context, id FonteID) (Fonte, bool, error) {
	var f Fonte
	var ativa bool
	err := s.pool.QueryRow(ctx, `SELECT id, mercado_id, url, filtro_nome_documento, ativa FROM fonte WHERE id = $1`, id).
		Scan(&f.ID, &f.MercadoID, &f.URL, &f.FiltroNomeDocumento, &ativa)
	if errors.Is(err, pgx.ErrNoRows) {
		return Fonte{}, false, nil
	}
	if err != nil {
		return Fonte{}, false, wrapPG(err)
	}
	f.Ativa = Bool(ativa)
	return f, true, nil
}

func (s *Catalog) ListFontes(ctx context.Context) ([]Fonte, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, mercado_id, url, filtro_nome_documento, ativa FROM fonte ORDER BY id`)
	if err != nil {
		return nil, wrapPG(err)
	}
	defer rows.Close()
	out := []Fonte{}
	for rows.Next() {
		var f Fonte
		var ativa bool
		if err := rows.Scan(&f.ID, &f.MercadoID, &f.URL, &f.FiltroNomeDocumento, &ativa); err != nil {
			return nil, wrapPG(err)
		}
		f.Ativa = Bool(ativa)
		out = append(out, f)
	}
	return out, wrapPG(rows.Err())
}

func (s *Catalog) DeleteFonte(ctx context.Context, id FonteID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM fonte WHERE id = $1`, id)
	if err != nil {
		return wrapPG(err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: fonte %s", ErrNotFound, id)
	}
	return nil
}

func (s *Catalog) SaveProduto(ctx context.Context, p Produto) error {
	if p.ID == "" {
		return fmt.Errorf("%w: produto id obrigatório", ErrInvalid)
	}
	cats := p.Categorias
	if cats == nil {
		cats = []string{}
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO produto (id, nome, nome_norm, categorias)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET nome = EXCLUDED.nome, nome_norm = EXCLUDED.nome_norm, categorias = EXCLUDED.categorias`,
		p.ID, p.Nome, p.NomeNorm, cats)
	return wrapPG(err)
}

func (s *Catalog) GetProduto(ctx context.Context, id ProdutoID) (Produto, bool, error) {
	var p Produto
	err := s.pool.QueryRow(ctx, `SELECT id, nome, nome_norm, categorias FROM produto WHERE id = $1`, id).
		Scan(&p.ID, &p.Nome, &p.NomeNorm, &p.Categorias)
	if errors.Is(err, pgx.ErrNoRows) {
		return Produto{}, false, nil
	}
	if p.Categorias == nil {
		p.Categorias = []string{}
	}
	return p, err == nil, wrapPG(err)
}

func (s *Catalog) GetProdutoByNomeNorm(ctx context.Context, nomeNorm string) (Produto, bool, error) {
	var p Produto
	err := s.pool.QueryRow(ctx, `SELECT id, nome, nome_norm, categorias FROM produto WHERE nome_norm = $1`, nomeNorm).
		Scan(&p.ID, &p.Nome, &p.NomeNorm, &p.Categorias)
	if errors.Is(err, pgx.ErrNoRows) {
		return Produto{}, false, nil
	}
	if p.Categorias == nil {
		p.Categorias = []string{}
	}
	return p, err == nil, wrapPG(err)
}

func (s *Catalog) ListProdutos(ctx context.Context) ([]Produto, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, nome, nome_norm, categorias FROM produto ORDER BY id`)
	if err != nil {
		return nil, wrapPG(err)
	}
	defer rows.Close()
	out := []Produto{}
	for rows.Next() {
		var p Produto
		if err := rows.Scan(&p.ID, &p.Nome, &p.NomeNorm, &p.Categorias); err != nil {
			return nil, wrapPG(err)
		}
		if p.Categorias == nil {
			p.Categorias = []string{}
		}
		out = append(out, p)
	}
	return out, wrapPG(rows.Err())
}

func (s *Catalog) DeleteProduto(ctx context.Context, id ProdutoID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM produto WHERE id = $1`, id)
	if err != nil {
		return wrapPG(err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: produto %s", ErrNotFound, id)
	}
	return nil
}

func (s *Catalog) SaveMarca(ctx context.Context, m Marca) error {
	if m.ID == "" {
		return fmt.Errorf("%w: marca id obrigatório", ErrInvalid)
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO marca (id, nome, nome_norm) VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET nome = EXCLUDED.nome, nome_norm = EXCLUDED.nome_norm`,
		m.ID, m.Nome, m.NomeNorm)
	return wrapPG(err)
}

func (s *Catalog) GetMarca(ctx context.Context, id MarcaID) (Marca, bool, error) {
	var m Marca
	err := s.pool.QueryRow(ctx, `SELECT id, nome, nome_norm FROM marca WHERE id = $1`, id).Scan(&m.ID, &m.Nome, &m.NomeNorm)
	if errors.Is(err, pgx.ErrNoRows) {
		return Marca{}, false, nil
	}
	return m, err == nil, wrapPG(err)
}

func (s *Catalog) GetMarcaByNomeNorm(ctx context.Context, nomeNorm string) (Marca, bool, error) {
	var m Marca
	err := s.pool.QueryRow(ctx, `SELECT id, nome, nome_norm FROM marca WHERE nome_norm = $1`, nomeNorm).Scan(&m.ID, &m.Nome, &m.NomeNorm)
	if errors.Is(err, pgx.ErrNoRows) {
		return Marca{}, false, nil
	}
	return m, err == nil, wrapPG(err)
}

func (s *Catalog) ListMarcas(ctx context.Context) ([]Marca, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, nome, nome_norm FROM marca ORDER BY id`)
	if err != nil {
		return nil, wrapPG(err)
	}
	defer rows.Close()
	out := []Marca{}
	for rows.Next() {
		var m Marca
		if err := rows.Scan(&m.ID, &m.Nome, &m.NomeNorm); err != nil {
			return nil, wrapPG(err)
		}
		out = append(out, m)
	}
	return out, wrapPG(rows.Err())
}

func (s *Catalog) DeleteMarca(ctx context.Context, id MarcaID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM marca WHERE id = $1`, id)
	if err != nil {
		return wrapPG(err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: marca %s", ErrNotFound, id)
	}
	return nil
}

func (s *Catalog) SaveDocumento(ctx context.Context, d Documento) error {
	if d.ID == "" {
		return fmt.Errorf("%w: documento id obrigatório", ErrInvalid)
	}
	atualizado := d.Atualizado
	if atualizado.IsZero() {
		atualizado = time.Time{}
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO documento (
			id, fonte_id, mercado_id, filename, dia, estado, fingerprint, conteudo_identico_a, ultimo_erro, atualizado)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (id) DO UPDATE SET
			fonte_id = EXCLUDED.fonte_id, mercado_id = EXCLUDED.mercado_id, filename = EXCLUDED.filename,
			dia = EXCLUDED.dia, estado = EXCLUDED.estado, fingerprint = EXCLUDED.fingerprint,
			conteudo_identico_a = EXCLUDED.conteudo_identico_a, ultimo_erro = EXCLUDED.ultimo_erro,
			atualizado = EXCLUDED.atualizado`,
		d.ID, d.FonteID, d.MercadoID, d.Filename, d.Dia, string(d.Estado), d.Fingerprint, d.ConteudoIdenticoA, d.UltimoErro, atualizado)
	return wrapPG(err)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDocumento(row rowScanner) (Documento, error) {
	var d Documento
	var estado string
	err := row.Scan(&d.ID, &d.FonteID, &d.MercadoID, &d.Filename, &d.Dia, &estado, &d.Fingerprint, &d.ConteudoIdenticoA, &d.UltimoErro, &d.Atualizado)
	d.Estado = EstadoDocumento(estado)
	return d, err
}

func (s *Catalog) GetDocumento(ctx context.Context, id DocumentoID) (Documento, bool, error) {
	d, err := scanDocumento(s.pool.QueryRow(ctx, `SELECT id, fonte_id, mercado_id, filename, dia, estado, fingerprint, conteudo_identico_a, ultimo_erro, atualizado FROM documento WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Documento{}, false, nil
	}
	return d, err == nil, wrapPG(err)
}

func (s *Catalog) GetDocumentoByIdentity(ctx context.Context, fonteID FonteID, filename, dia string) (Documento, bool, error) {
	d, err := scanDocumento(s.pool.QueryRow(ctx, `SELECT id, fonte_id, mercado_id, filename, dia, estado, fingerprint, conteudo_identico_a, ultimo_erro, atualizado FROM documento WHERE fonte_id = $1 AND filename = $2 AND dia = $3`, fonteID, filename, dia))
	if errors.Is(err, pgx.ErrNoRows) {
		return Documento{}, false, nil
	}
	return d, err == nil, wrapPG(err)
}

func (s *Catalog) ListDocumentos(ctx context.Context) ([]Documento, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, fonte_id, mercado_id, filename, dia, estado, fingerprint, conteudo_identico_a, ultimo_erro, atualizado FROM documento ORDER BY id`)
	if err != nil {
		return nil, wrapPG(err)
	}
	defer rows.Close()
	out := []Documento{}
	for rows.Next() {
		d, err := scanDocumento(rows)
		if err != nil {
			return nil, wrapPG(err)
		}
		out = append(out, d)
	}
	return out, wrapPG(rows.Err())
}

func (s *Catalog) DeleteDocumento(ctx context.Context, id DocumentoID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return wrapPG(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var ofertaIDs []OfertaID
	rows, err := tx.Query(ctx, `SELECT oferta_id FROM documento_oferta WHERE documento_id = $1`, id)
	if err != nil {
		return wrapPG(err)
	}
	for rows.Next() {
		var oid OfertaID
		if err := rows.Scan(&oid); err != nil {
			rows.Close()
			return wrapPG(err)
		}
		ofertaIDs = append(ofertaIDs, oid)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return wrapPG(err)
	}

	tag, err := tx.Exec(ctx, `DELETE FROM documento WHERE id = $1`, id)
	if err != nil {
		return wrapPG(err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: documento %s", ErrNotFound, id)
	}
	for _, oid := range ofertaIDs {
		var n int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM documento_oferta WHERE oferta_id = $1`, oid).Scan(&n); err != nil {
			return wrapPG(err)
		}
		if n == 0 {
			if _, err := tx.Exec(ctx, `DELETE FROM oferta WHERE id = $1`, oid); err != nil {
				return wrapPG(err)
			}
		}
	}
	return wrapPG(tx.Commit(ctx))
}

func (s *Catalog) EarliestDia(ctx context.Context, fonteID FonteID, filename string) (string, bool, error) {
	var dia string
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(MIN(dia), '') FROM documento WHERE fonte_id = $1 AND filename = $2`, fonteID, filename).Scan(&dia)
	if errors.Is(err, pgx.ErrNoRows) || dia == "" {
		return "", false, wrapPG(errIfNoRows(err))
	}
	return dia, err == nil, wrapPG(err)
}

func errIfNoRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

func (s *Catalog) ListDias(ctx context.Context, fonteID FonteID, filename string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT dia FROM documento WHERE fonte_id = $1 AND filename = $2 ORDER BY dia`, fonteID, filename)
	if err != nil {
		return nil, wrapPG(err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var dia string
		if err := rows.Scan(&dia); err != nil {
			return nil, wrapPG(err)
		}
		out = append(out, dia)
	}
	return out, wrapPG(rows.Err())
}

func (s *Catalog) SaveOferta(ctx context.Context, o Oferta, documentoIDs []DocumentoID) error {
	if o.ID == "" {
		return fmt.Errorf("%w: oferta id obrigatório", ErrInvalid)
	}
	if len(documentoIDs) == 0 {
		return fmt.Errorf("%w: oferta exige ao menos um documento", ErrInvalid)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return wrapPG(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := s.saveOfertaTx(ctx, tx, o, documentoIDs); err != nil {
		return err
	}
	return wrapPG(tx.Commit(ctx))
}

func (s *Catalog) saveOfertaTx(ctx context.Context, tx pgx.Tx, o Oferta, documentoIDs []DocumentoID) error {
	chave := ChaveUnicaOferta(o)
	var existingID OfertaID
	err := tx.QueryRow(ctx, `SELECT id FROM oferta WHERE chave_unica = $1`, chave).Scan(&existingID)
	if err == nil && existingID != o.ID {
		return fmt.Errorf("%w: chave unica de oferta já existe", ErrConflict)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return wrapPG(err)
	}
	if o.DocumentoID == "" {
		o.DocumentoID = documentoIDs[0]
	}
	if err := upsertOferta(ctx, tx, o, chave); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM documento_oferta WHERE oferta_id = $1`, o.ID); err != nil {
		return wrapPG(err)
	}
	for _, docID := range documentoIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO documento_oferta (documento_id, oferta_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, docID, o.ID); err != nil {
			return wrapPG(err)
		}
	}
	return nil
}

func upsertOferta(ctx context.Context, tx pgx.Tx, o Oferta, chave string) error {
	promo, err := jsonNull(o.Promocao)
	if err != nil {
		return err
	}
	comp, err := jsonNull(o.Comparativo)
	if err != nil {
		return err
	}
	quants := o.Quantidades
	if quants == nil {
		quants = []float64{}
	}
	_, err = tx.Exec(ctx, `INSERT INTO oferta (
			id, produto_id, marca_id, mercado_id, valor, quantidades, medida,
			data_inicio, data_expiracao, origem_data_inicio, origem_data_expiracao,
			promocao, comparativo, chave_unica, documento_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (id) DO UPDATE SET
			produto_id = EXCLUDED.produto_id, marca_id = EXCLUDED.marca_id, mercado_id = EXCLUDED.mercado_id,
			valor = EXCLUDED.valor, quantidades = EXCLUDED.quantidades, medida = EXCLUDED.medida,
			data_inicio = EXCLUDED.data_inicio, data_expiracao = EXCLUDED.data_expiracao,
			origem_data_inicio = EXCLUDED.origem_data_inicio, origem_data_expiracao = EXCLUDED.origem_data_expiracao,
			promocao = EXCLUDED.promocao, comparativo = EXCLUDED.comparativo,
			chave_unica = EXCLUDED.chave_unica, documento_id = EXCLUDED.documento_id`,
		o.ID, o.ProdutoID, o.MarcaID, o.MercadoID, o.Valor, quants, string(o.Medida),
		o.DataInicio, o.DataExpiracao, string(o.OrigemDataInicio), string(o.OrigemDataExpiracao),
		promo, comp, chave, o.DocumentoID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "chave_unica") {
			return fmt.Errorf("%w: chave unica de oferta já existe", ErrConflict)
		}
	}
	return wrapPG(err)
}

func jsonNull(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	switch t := v.(type) {
	case *Promocao:
		if t == nil {
			return nil, nil
		}
	case *Comparativo:
		if t == nil {
			return nil, nil
		}
	}
	return json.Marshal(v)
}

func (s *Catalog) GetOferta(ctx context.Context, id OfertaID) (Oferta, bool, error) {
	o, err := scanOferta(s.pool.QueryRow(ctx, ofertaSelect+" WHERE id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Oferta{}, false, nil
	}
	return o, err == nil, wrapPG(err)
}

func (s *Catalog) GetOfertaByUniq(ctx context.Context, chave string) (Oferta, bool, error) {
	o, err := scanOferta(s.pool.QueryRow(ctx, ofertaSelect+" WHERE chave_unica = $1", chave))
	if errors.Is(err, pgx.ErrNoRows) {
		return Oferta{}, false, nil
	}
	return o, err == nil, wrapPG(err)
}

const ofertaSelect = `SELECT id, produto_id, marca_id, mercado_id, valor, quantidades, medida,
	data_inicio, data_expiracao, origem_data_inicio, origem_data_expiracao,
	promocao, comparativo, documento_id FROM oferta`

func scanOferta(row rowScanner) (Oferta, error) {
	var o Oferta
	var medida, origemIni, origemFim string
	var promo, comp []byte
	err := row.Scan(&o.ID, &o.ProdutoID, &o.MarcaID, &o.MercadoID, &o.Valor, &o.Quantidades, &medida,
		&o.DataInicio, &o.DataExpiracao, &origemIni, &origemFim, &promo, &comp, &o.DocumentoID)
	o.Medida = Medida(medida)
	o.OrigemDataInicio = OrigemData(origemIni)
	o.OrigemDataExpiracao = OrigemData(origemFim)
	if len(promo) > 0 {
		var p Promocao
		if err := json.Unmarshal(promo, &p); err != nil {
			return Oferta{}, err
		}
		o.Promocao = &p
	}
	if len(comp) > 0 {
		var c Comparativo
		if err := json.Unmarshal(comp, &c); err != nil {
			return Oferta{}, err
		}
		o.Comparativo = &c
	}
	return o, err
}

func (s *Catalog) ListOfertas(ctx context.Context) ([]Oferta, error) {
	rows, err := s.pool.Query(ctx, ofertaSelect+" ORDER BY id")
	if err != nil {
		return nil, wrapPG(err)
	}
	defer rows.Close()
	out := []Oferta{}
	for rows.Next() {
		o, err := scanOferta(rows)
		if err != nil {
			return nil, wrapPG(err)
		}
		out = append(out, o)
	}
	return out, wrapPG(rows.Err())
}

func (s *Catalog) DeleteOferta(ctx context.Context, id OfertaID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM oferta WHERE id = $1`, id)
	if err != nil {
		return wrapPG(err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: oferta %s", ErrNotFound, id)
	}
	return nil
}

func (s *Catalog) SaveOfertasForDocumento(ctx context.Context, documentoID DocumentoID, ofertas []Oferta) error {
	if ofertas == nil {
		ofertas = []Oferta{}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return wrapPG(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	prevIDs := map[OfertaID]struct{}{}
	rows, err := tx.Query(ctx, `SELECT oferta_id FROM documento_oferta WHERE documento_id = $1`, documentoID)
	if err != nil {
		return wrapPG(err)
	}
	for rows.Next() {
		var id OfertaID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return wrapPG(err)
		}
		prevIDs[id] = struct{}{}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return wrapPG(err)
	}

	nextIDs := map[OfertaID]struct{}{}
	for _, o := range ofertas {
		chave := ChaveUnicaOferta(o)
		existing, err := scanOferta(tx.QueryRow(ctx, ofertaSelect+" WHERE chave_unica = $1", chave))
		if err == nil {
			o = existing
		} else if errors.Is(err, pgx.ErrNoRows) {
			if o.ID == "" {
				return fmt.Errorf("%w: oferta id obrigatório para criar entidade", ErrInvalid)
			}
			if o.DocumentoID == "" {
				o.DocumentoID = documentoID
			}
			if err := upsertOferta(ctx, tx, o, chave); err != nil {
				return err
			}
		} else {
			return wrapPG(err)
		}
		nextIDs[o.ID] = struct{}{}
		if _, err := tx.Exec(ctx, `INSERT INTO documento_oferta (documento_id, oferta_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, documentoID, o.ID); err != nil {
			return wrapPG(err)
		}
	}

	for id := range prevIDs {
		if _, keep := nextIDs[id]; keep {
			continue
		}
		if _, err := tx.Exec(ctx, `DELETE FROM documento_oferta WHERE documento_id = $1 AND oferta_id = $2`, documentoID, id); err != nil {
			return wrapPG(err)
		}
		var n int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM documento_oferta WHERE oferta_id = $1`, id).Scan(&n); err != nil {
			return wrapPG(err)
		}
		if n == 0 {
			if _, err := tx.Exec(ctx, `DELETE FROM oferta WHERE id = $1`, id); err != nil {
				return wrapPG(err)
			}
		}
	}
	return wrapPG(tx.Commit(ctx))
}

func (s *Catalog) ListOfertasByDocumento(ctx context.Context, documentoID DocumentoID) ([]Oferta, error) {
	rows, err := s.pool.Query(ctx, ofertaSelect+` WHERE id IN (SELECT oferta_id FROM documento_oferta WHERE documento_id = $1) ORDER BY id`, documentoID)
	if err != nil {
		return nil, wrapPG(err)
	}
	defer rows.Close()
	out := []Oferta{}
	for rows.Next() {
		o, err := scanOferta(rows)
		if err != nil {
			return nil, wrapPG(err)
		}
		out = append(out, o)
	}
	return out, wrapPG(rows.Err())
}

func (s *Catalog) ListDocumentoIDsByProduto(ctx context.Context, produtoID ProdutoID) ([]DocumentoID, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT d.oferta_id, d.documento_id
		FROM documento_oferta d
		JOIN oferta o ON o.id = d.oferta_id
		WHERE o.produto_id = $1
		ORDER BY d.documento_id`, produtoID)
	if err != nil {
		return nil, wrapPG(err)
	}
	defer rows.Close()
	out := []DocumentoID{}
	seen := map[DocumentoID]struct{}{}
	for rows.Next() {
		var oid OfertaID
		var did DocumentoID
		if err := rows.Scan(&oid, &did); err != nil {
			return nil, wrapPG(err)
		}
		if _, ok := seen[did]; ok {
			continue
		}
		seen[did] = struct{}{}
		out = append(out, did)
	}
	return out, wrapPG(rows.Err())
}

func (s *Catalog) SaveFalhasDocumento(ctx context.Context, documentoID DocumentoID, falhas []FalhaExtracao) error {
	if falhas == nil {
		falhas = []FalhaExtracao{}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return wrapPG(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM falha_extracao WHERE documento_id = $1`, documentoID); err != nil {
		return wrapPG(err)
	}
	for i, f := range falhas {
		cand, err := json.Marshal(f.Candidato)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO falha_extracao (documento_id, pos, codigo, detalhe, candidato) VALUES ($1,$2,$3,$4,$5)`,
			documentoID, i, f.Codigo, f.Detalhe, cand); err != nil {
			return wrapPG(err)
		}
	}
	return wrapPG(tx.Commit(ctx))
}

func (s *Catalog) GetFalhasDocumento(ctx context.Context, documentoID DocumentoID) ([]FalhaExtracao, error) {
	rows, err := s.pool.Query(ctx, `SELECT codigo, detalhe, candidato FROM falha_extracao WHERE documento_id = $1 ORDER BY pos`, documentoID)
	if err != nil {
		return nil, wrapPG(err)
	}
	defer rows.Close()
	out := []FalhaExtracao{}
	for rows.Next() {
		var f FalhaExtracao
		var cand []byte
		if err := rows.Scan(&f.Codigo, &f.Detalhe, &cand); err != nil {
			return nil, wrapPG(err)
		}
		if len(cand) > 0 {
			if err := json.Unmarshal(cand, &f.Candidato); err != nil {
				return nil, err
			}
		}
		out = append(out, f)
	}
	return out, wrapPG(rows.Err())
}

func (s *Catalog) DeleteFalhasDocumento(ctx context.Context, documentoID DocumentoID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM falha_extracao WHERE documento_id = $1`, documentoID)
	return wrapPG(err)
}

func (s *Catalog) SaveUsoExtrator(ctx context.Context, uso UsoExtrator) error {
	if uso.DocumentoID == "" || uso.Tentativa == "" {
		return fmt.Errorf("%w: uso-extrator documentoId e tentativa obrigatórios", ErrInvalid)
	}
	paginas, err := jsonNullSlice(uso.Paginas)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO uso_extrator (
			documento_id, tentativa, artefato_path, provider, model, prompt_tokens, cache_tokens, output_tokens, paginas)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (documento_id, tentativa) DO UPDATE SET
			artefato_path = EXCLUDED.artefato_path, provider = EXCLUDED.provider, model = EXCLUDED.model,
			prompt_tokens = EXCLUDED.prompt_tokens, cache_tokens = EXCLUDED.cache_tokens,
			output_tokens = EXCLUDED.output_tokens, paginas = EXCLUDED.paginas`,
		uso.DocumentoID, uso.Tentativa, uso.ArtefatoPath, uso.Provider, uso.Model,
		uso.PromptTokens, uso.CacheTokens, uso.OutputTokens, paginas)
	return wrapPG(err)
}

func jsonNullSlice(v []UsoExtratorPagina) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

func (s *Catalog) GetUsoExtrator(ctx context.Context, documentoID DocumentoID, tentativa string) (UsoExtrator, bool, error) {
	u, err := scanUso(s.pool.QueryRow(ctx, usoSelect+" WHERE documento_id = $1 AND tentativa = $2", documentoID, tentativa))
	if errors.Is(err, pgx.ErrNoRows) {
		return UsoExtrator{}, false, nil
	}
	return u, err == nil, wrapPG(err)
}

const usoSelect = `SELECT documento_id, tentativa, artefato_path, provider, model, prompt_tokens, cache_tokens, output_tokens, paginas FROM uso_extrator`

func scanUso(row rowScanner) (UsoExtrator, error) {
	var u UsoExtrator
	var paginas []byte
	err := row.Scan(&u.DocumentoID, &u.Tentativa, &u.ArtefatoPath, &u.Provider, &u.Model, &u.PromptTokens, &u.CacheTokens, &u.OutputTokens, &paginas)
	if len(paginas) > 0 {
		if err := json.Unmarshal(paginas, &u.Paginas); err != nil {
			return UsoExtrator{}, err
		}
	}
	return u, err
}

func (s *Catalog) ListUsosExtrator(ctx context.Context, documentoID DocumentoID) ([]UsoExtrator, error) {
	var rows pgx.Rows
	var err error
	if documentoID != "" {
		rows, err = s.pool.Query(ctx, usoSelect+" WHERE documento_id = $1 ORDER BY tentativa", documentoID)
	} else {
		rows, err = s.pool.Query(ctx, usoSelect+" ORDER BY documento_id, tentativa")
	}
	if err != nil {
		return nil, wrapPG(err)
	}
	defer rows.Close()
	out := []UsoExtrator{}
	for rows.Next() {
		u, err := scanUso(rows)
		if err != nil {
			return nil, wrapPG(err)
		}
		out = append(out, u)
	}
	return out, wrapPG(rows.Err())
}

func (s *Catalog) DeleteUsoExtrator(ctx context.Context, documentoID DocumentoID, tentativa string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM uso_extrator WHERE documento_id = $1 AND tentativa = $2`, documentoID, tentativa)
	return wrapPG(err)
}

func (s *Catalog) Esgotado(ctx context.Context, provider, dia string) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM extrator_cota WHERE provider = $1 AND dia = $2`, provider, dia).Scan(&n)
	return n > 0, wrapPG(err)
}

func (s *Catalog) MarcarEsgotado(ctx context.Context, provider, dia string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO extrator_cota (provider, dia) VALUES ($1, $2) ON CONFLICT DO NOTHING`, provider, dia)
	return wrapPG(err)
}

func (s *Catalog) TruncateAll(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `TRUNCATE
		extrator_cota, operacao_pipeline, uso_extrator, falha_extracao,
		documento_oferta, oferta, documento, fonte, produto, marca, mercado
		RESTART IDENTITY CASCADE`)
	return wrapPG(err)
}
