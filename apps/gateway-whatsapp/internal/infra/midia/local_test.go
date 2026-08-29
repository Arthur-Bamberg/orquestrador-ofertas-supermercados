package midia_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/midia"
)

func TestLocal_guardaELeBytes(t *testing.T) {
	store := midia.Local{Root: t.TempDir()}
	path, err := store.Guardar(context.Background(), "msg-1", domain.MidiaBytes{
		Tipo:     domain.MidiaAudio,
		Filename: "voz.ogg",
		MIME:     "audio/ogg",
		Conteudo: []byte("ogg"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(path) != ".ogg" {
		t.Fatalf("path=%s", path)
	}
	got, err := store.Ler(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ogg" {
		t.Fatalf("got=%q", got)
	}
}

func TestLocal_rejeitaPathForaDaRaiz(t *testing.T) {
	store := midia.Local{Root: t.TempDir()}
	_, err := store.Ler(context.Background(), "../secret")
	if err == nil {
		t.Fatal("esperava rejeitar path fora da raiz")
	}
}
