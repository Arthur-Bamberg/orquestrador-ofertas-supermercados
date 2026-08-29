package redismigrate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

// CopyFromRedis copies catalog and operational keys from a Redis/Upstash client into Postgres.
func CopyFromRedis(ctx context.Context, src *Client, dst *store.Catalog) error {
	if err := copyJSONSet(ctx, src, dst, "mercados", "mercado:", func(raw string) error {
		var m store.Mercado
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return err
		}
		return dst.SaveMercado(ctx, m)
	}); err != nil {
		return err
	}
	if err := copyJSONSet(ctx, src, dst, "fontes", "fonte:", func(raw string) error {
		var f store.Fonte
		if err := json.Unmarshal([]byte(raw), &f); err != nil {
			return err
		}
		return dst.SaveFonte(ctx, f)
	}); err != nil {
		return err
	}
	if err := copyJSONSet(ctx, src, dst, "produtos", "produto:", func(raw string) error {
		var p store.Produto
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			return err
		}
		return dst.SaveProduto(ctx, p)
	}); err != nil {
		return err
	}
	if err := copyJSONSet(ctx, src, dst, "marcas", "marca:", func(raw string) error {
		var m store.Marca
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return err
		}
		return dst.SaveMarca(ctx, m)
	}); err != nil {
		return err
	}
	if err := copyJSONSet(ctx, src, dst, "documentos", "documento:", func(raw string) error {
		var d store.Documento
		if err := json.Unmarshal([]byte(raw), &d); err != nil {
			return err
		}
		return dst.SaveDocumento(ctx, d)
	}); err != nil {
		return err
	}

	ofertaIDs, err := src.SMembers(ctx, "ofertas")
	if err != nil {
		return err
	}
	if len(ofertaIDs) == 0 {
		keys, err := src.Keys(ctx, "oferta:*")
		if err != nil {
			return err
		}
		for _, key := range keys {
			if strings.HasPrefix(key, "oferta:uniq:") || strings.HasPrefix(key, "oferta:documentos:") {
				continue
			}
			ofertaIDs = append(ofertaIDs, strings.TrimPrefix(key, "oferta:"))
		}
	}
	for _, id := range ofertaIDs {
		raw, ok, err := src.Get(ctx, "oferta:"+id)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		var o store.Oferta
		if err := json.Unmarshal([]byte(raw), &o); err != nil {
			return err
		}
		docs, err := src.SMembers(ctx, "oferta:documentos:"+id)
		if err != nil {
			return err
		}
		var docIDs []store.DocumentoID
		for _, d := range docs {
			docIDs = append(docIDs, store.DocumentoID(d))
		}
		if len(docIDs) == 0 && o.DocumentoID != "" {
			docIDs = []store.DocumentoID{o.DocumentoID}
		}
		if len(docIDs) == 0 {
			return fmt.Errorf("oferta %s: sem Documento associado", id)
		}
		if err := dst.SaveOferta(ctx, o, docIDs); err != nil {
			return fmt.Errorf("oferta %s: %w", id, err)
		}
	}

	docIDs, err := entityIDs(ctx, src, "documentos", "documento:")
	if err != nil {
		return err
	}
	for _, id := range docIDs {
		raw, ok, err := src.Get(ctx, "falhas:documento:"+id)
		if err != nil {
			return err
		}
		if !ok || raw == "" || raw == "null" {
			continue
		}
		var falhas []store.FalhaExtracao
		if err := json.Unmarshal([]byte(raw), &falhas); err != nil {
			return err
		}
		if err := dst.SaveFalhasDocumento(ctx, store.DocumentoID(id), falhas); err != nil {
			return err
		}
	}

	usoKeys, err := src.SMembers(ctx, "usos-extrator")
	if err != nil {
		return err
	}
	if len(usoKeys) == 0 {
		usoKeys, err = src.Keys(ctx, "uso-extrator:*:*")
		if err != nil {
			return err
		}
	}
	for _, key := range usoKeys {
		if strings.HasPrefix(key, "uso-extrator:documento:") {
			continue
		}
		raw, ok, err := src.Get(ctx, key)
		if err != nil || !ok {
			continue
		}
		var uso store.UsoExtrator
		if err := json.Unmarshal([]byte(raw), &uso); err != nil {
			return err
		}
		if err := dst.SaveUsoExtrator(ctx, uso); err != nil {
			return err
		}
	}

	history, okHist, err := src.Get(ctx, "ofertas-api:ops:history")
	if err != nil {
		return err
	}
	var opIDs []string
	if okHist && history != "" && history != "null" {
		_ = json.Unmarshal([]byte(history), &opIDs)
	}
	if len(opIDs) == 0 {
		keys, err := src.Keys(ctx, "ofertas-api:ops:*")
		if err != nil {
			return err
		}
		for _, key := range keys {
			if key == "ofertas-api:ops:queue" || key == "ofertas-api:ops:history" {
				continue
			}
			opIDs = append(opIDs, strings.TrimPrefix(key, "ofertas-api:ops:"))
		}
	}
	for _, id := range opIDs {
		raw, ok, err := src.Get(ctx, "ofertas-api:ops:"+id)
		if err != nil || !ok {
			continue
		}
		var op store.OperacaoPipeline
		if err := json.Unmarshal([]byte(raw), &op); err != nil {
			return err
		}
		if err := dst.UpsertOperacao(ctx, op); err != nil {
			return fmt.Errorf("operacao %s: %w", id, err)
		}
	}

	cotaKeys, err := src.Keys(ctx, "ofertas-scraper:extrator:*:esgotado:*")
	if err != nil {
		return err
	}
	for _, key := range cotaKeys {
		parts := strings.Split(key, ":")
		if len(parts) < 5 {
			continue
		}
		provider := parts[2]
		dia := parts[4]
		if err := dst.MarcarEsgotado(ctx, provider, dia); err != nil {
			return err
		}
	}
	return nil
}

func copyJSONSet(ctx context.Context, src *Client, _ *store.Catalog, setKey, prefix string, save func(string) error) error {
	ids, err := entityIDs(ctx, src, setKey, prefix)
	if err != nil {
		return err
	}
	for _, id := range ids {
		raw, ok, err := src.Get(ctx, prefix+id)
		if err != nil || !ok {
			continue
		}
		if err := save(raw); err != nil {
			return fmt.Errorf("%s%s: %w", prefix, id, err)
		}
	}
	return nil
}

func entityIDs(ctx context.Context, src *Client, setKey, prefix string) ([]string, error) {
	ids, err := src.SMembers(ctx, setKey)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		keys, err := src.Keys(ctx, prefix+"*")
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			if strings.Count(key, ":") != 1 {
				continue
			}
			ids = append(ids, strings.TrimPrefix(key, prefix))
		}
	}
	out := ids[:0]
	for _, id := range ids {
		if strings.HasPrefix(id, "norm:") {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}
