package whatsapp_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/whatsapp"
	_ "modernc.org/sqlite"
)

func TestLigar_ativaWALNaSessao(t *testing.T) {
	path := filepath.Join(t.TempDir(), "whatsmeow.db")
	cli, err := whatsapp.Ligar(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cli.Disconnect)

	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(mode, "wal") {
		t.Fatalf("journal_mode=%q, want wal (sem WAL o whatsmeow toma SQLITE_BUSY após o pairing)", mode)
	}
}
