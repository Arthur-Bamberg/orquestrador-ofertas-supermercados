CREATE TABLE mercado (
    id TEXT PRIMARY KEY,
    nome TEXT NOT NULL
);

CREATE TABLE fonte (
    id TEXT PRIMARY KEY,
    mercado_id TEXT NOT NULL REFERENCES mercado (id) ON DELETE RESTRICT,
    url TEXT NOT NULL,
    filtro_nome_documento TEXT NOT NULL DEFAULT ''
);

CREATE TABLE produto (
    id TEXT PRIMARY KEY,
    nome TEXT NOT NULL,
    nome_norm TEXT NOT NULL DEFAULT '',
    categorias TEXT[] NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX produto_nome_norm ON produto (nome_norm) WHERE nome_norm <> '';

CREATE TABLE marca (
    id TEXT PRIMARY KEY,
    nome TEXT NOT NULL,
    nome_norm TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX marca_nome_norm ON marca (nome_norm) WHERE nome_norm <> '';

CREATE TABLE documento (
    id TEXT PRIMARY KEY,
    fonte_id TEXT NOT NULL REFERENCES fonte (id) ON DELETE RESTRICT,
    mercado_id TEXT NOT NULL REFERENCES mercado (id) ON DELETE RESTRICT,
    filename TEXT NOT NULL,
    dia TEXT NOT NULL,
    estado TEXT NOT NULL DEFAULT '',
    fingerprint TEXT NOT NULL DEFAULT '',
    conteudo_identico_a TEXT REFERENCES documento (id) ON DELETE SET NULL,
    ultimo_erro TEXT NOT NULL DEFAULT '',
    atualizado TIMESTAMPTZ NOT NULL DEFAULT TIMESTAMPTZ '1970-01-01 00:00:00+00',
    UNIQUE (fonte_id, filename, dia)
);

CREATE TABLE oferta (
    id TEXT PRIMARY KEY,
    produto_id TEXT NOT NULL REFERENCES produto (id) ON DELETE RESTRICT,
    marca_id TEXT REFERENCES marca (id) ON DELETE RESTRICT,
    mercado_id TEXT NOT NULL REFERENCES mercado (id) ON DELETE RESTRICT,
    valor DOUBLE PRECISION NOT NULL,
    quantidades DOUBLE PRECISION[] NOT NULL DEFAULT '{}',
    medida TEXT NOT NULL DEFAULT '',
    data_inicio TEXT NOT NULL DEFAULT '',
    data_expiracao TEXT NOT NULL DEFAULT '',
    origem_data_inicio TEXT NOT NULL DEFAULT '',
    origem_data_expiracao TEXT NOT NULL DEFAULT '',
    promocao JSONB,
    comparativo JSONB,
    chave_unica TEXT NOT NULL UNIQUE,
    documento_id TEXT NOT NULL DEFAULT ''
);

CREATE INDEX oferta_produto_id ON oferta (produto_id);

CREATE TABLE documento_oferta (
    documento_id TEXT NOT NULL REFERENCES documento (id) ON DELETE CASCADE,
    oferta_id TEXT NOT NULL REFERENCES oferta (id) ON DELETE CASCADE,
    PRIMARY KEY (documento_id, oferta_id)
);

CREATE INDEX documento_oferta_oferta_id ON documento_oferta (oferta_id);

CREATE TABLE falha_extracao (
    id BIGSERIAL PRIMARY KEY,
    documento_id TEXT NOT NULL REFERENCES documento (id) ON DELETE CASCADE,
    pos INTEGER NOT NULL,
    codigo TEXT NOT NULL,
    detalhe TEXT NOT NULL DEFAULT '',
    candidato JSONB NOT NULL DEFAULT '{}'::jsonb,
    UNIQUE (documento_id, pos)
);

CREATE TABLE uso_extrator (
    documento_id TEXT NOT NULL REFERENCES documento (id) ON DELETE CASCADE,
    tentativa TEXT NOT NULL,
    artefato_path TEXT NOT NULL DEFAULT '',
    provider TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    prompt_tokens BIGINT NOT NULL DEFAULT 0,
    cache_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    paginas JSONB,
    PRIMARY KEY (documento_id, tentativa)
);

CREATE TABLE operacao_pipeline (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    status TEXT NOT NULL,
    fonte_id TEXT NOT NULL DEFAULT '',
    documento_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    log_snippet TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    queue_pos BIGINT
);

CREATE INDEX operacao_pipeline_pending_queue ON operacao_pipeline (queue_pos)
    WHERE status = 'pending';

CREATE SEQUENCE operacao_pipeline_queue_seq;

CREATE TABLE extrator_cota (
    provider TEXT NOT NULL,
    dia TEXT NOT NULL,
    PRIMARY KEY (provider, dia)
);
