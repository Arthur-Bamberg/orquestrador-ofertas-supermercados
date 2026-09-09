ALTER TABLE coleta ADD COLUMN termo TEXT NOT NULL DEFAULT '';
UPDATE coleta SET termo = produto_id WHERE termo = '';
ALTER TABLE coleta DROP CONSTRAINT coleta_produto_id_mercado_id_dia_key;
ALTER TABLE coleta DROP CONSTRAINT coleta_produto_id_fkey;
ALTER TABLE coleta DROP COLUMN produto_id;
ALTER TABLE coleta ADD CONSTRAINT coleta_termo_mercado_id_dia_key UNIQUE (termo, mercado_id, dia);
ALTER TABLE coleta ALTER COLUMN termo DROP DEFAULT;
