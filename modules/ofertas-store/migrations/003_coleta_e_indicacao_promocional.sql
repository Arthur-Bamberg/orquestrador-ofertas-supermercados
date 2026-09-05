ALTER TABLE oferta
    ADD COLUMN indicacao_promocional BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE coleta (
    id TEXT PRIMARY KEY,
    produto_id TEXT NOT NULL REFERENCES produto (id) ON DELETE RESTRICT,
    mercado_id TEXT NOT NULL REFERENCES mercado (id) ON DELETE RESTRICT,
    dia TEXT NOT NULL,
    estado TEXT NOT NULL DEFAULT '',
    ultimo_erro TEXT NOT NULL DEFAULT '',
    atualizado TIMESTAMPTZ NOT NULL DEFAULT TIMESTAMPTZ '1970-01-01 00:00:00+00',
    UNIQUE (produto_id, mercado_id, dia)
);

CREATE TABLE coleta_oferta (
    coleta_id TEXT NOT NULL REFERENCES coleta (id) ON DELETE CASCADE,
    oferta_id TEXT NOT NULL REFERENCES oferta (id) ON DELETE CASCADE,
    PRIMARY KEY (coleta_id, oferta_id)
);

CREATE INDEX coleta_oferta_oferta_id ON coleta_oferta (oferta_id);
