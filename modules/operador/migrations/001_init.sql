CREATE SCHEMA IF NOT EXISTS backoffice;

CREATE TABLE backoffice.operador (
  id TEXT PRIMARY KEY,
  nome TEXT NOT NULL UNIQUE,
  senha_hash TEXT NOT NULL,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE backoffice.identificacao (
  id TEXT PRIMARY KEY,
  operador_id TEXT NOT NULL REFERENCES backoffice.operador(id) ON DELETE CASCADE,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX backoffice_identificacao_operador ON backoffice.identificacao (operador_id);
