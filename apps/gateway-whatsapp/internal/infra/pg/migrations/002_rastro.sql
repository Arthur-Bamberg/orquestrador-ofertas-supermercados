ALTER TABLE whatsapp.contato ADD COLUMN jid_lid TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX contato_jid_lid ON whatsapp.contato (jid_lid) WHERE jid_lid <> '';

ALTER TABLE whatsapp.conversa ADD COLUMN jid_lid TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX conversa_jid_lid ON whatsapp.conversa (jid_lid) WHERE jid_lid <> '';

ALTER TABLE whatsapp.mensagem ADD COLUMN origem TEXT NOT NULL DEFAULT 'vivo';
ALTER TABLE whatsapp.mensagem ADD COLUMN push_name TEXT NOT NULL DEFAULT '';
ALTER TABLE whatsapp.mensagem ADD COLUMN tipo TEXT NOT NULL DEFAULT '';
ALTER TABLE whatsapp.mensagem ADD COLUMN payload JSONB NOT NULL DEFAULT '{}'::jsonb;
