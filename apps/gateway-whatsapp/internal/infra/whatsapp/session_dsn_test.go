package whatsapp

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionDSN_aplicaBusyTimeout(t *testing.T) {
	db, err := sql.Open("sqlite", sessionDSN(filepath.Join(t.TempDir(), "s.db")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var timeout int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil {
		t.Fatal(err)
	}
	if timeout != sqliteBusyTimeoutMS {
		t.Fatalf("busy_timeout=%d, want %d (sem espera o whatsmeow toma SQLITE_BUSY)", timeout, sqliteBusyTimeoutMS)
	}
}

func TestSessionDSN_segundaEscritaEsperaOLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.db")
	db, err := sql.Open("sqlite", sessionDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}

	conn1, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn1.Close() })

	tx, err := conn1.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT INTO t (id) VALUES (1)`); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := db.Exec(`INSERT INTO t (id) VALUES (2)`)
		done <- err
	}()

	select {
	case err := <-done:
		t.Fatalf("segunda escrita deveria esperar o lock, retornou cedo: %v", err)
	case <-time.After(150 * time.Millisecond):
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("segunda escrita não concluiu após o commit")
	}
}
