CREATE TABLE whatsapp.contato (
    id TEXT PRIMARY KEY,
    jid TEXT NOT NULL UNIQUE
);

CREATE TABLE whatsapp.conversa (
    id TEXT PRIMARY KEY,
    jid TEXT NOT NULL UNIQUE,
    tipo TEXT NOT NULL
);

CREATE TABLE whatsapp.mensagem (
    id TEXT PRIMARY KEY,
    conversa_id TEXT NOT NULL REFERENCES whatsapp.conversa (id),
    contato_id TEXT NOT NULL REFERENCES whatsapp.contato (id),
    direcao TEXT NOT NULL,
    corpo TEXT NOT NULL DEFAULT '',
    provedor_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    midia_tipo TEXT NOT NULL DEFAULT '',
    midia_path TEXT NOT NULL DEFAULT '',
    midia_filename TEXT NOT NULL DEFAULT '',
    midia_mime TEXT NOT NULL DEFAULT '',
    criado_em TIMESTAMPTZ NOT NULL DEFAULT TIMESTAMPTZ '1970-01-01 00:00:00+00'
);

CREATE UNIQUE INDEX mensagem_provedor_id ON whatsapp.mensagem (provedor_id) WHERE provedor_id <> '';
CREATE INDEX mensagem_conversa_id ON whatsapp.mensagem (conversa_id);
