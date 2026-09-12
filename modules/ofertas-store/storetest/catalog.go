package storetest

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

var (
	pgOnce   sync.Once
	pgErr    error
	pgDSN    string
	pgServer *embeddedpostgres.EmbeddedPostgres
)

func DSN(t *testing.T) string {
	t.Helper()
	start(t)
	return pgDSN
}

func New(t *testing.T) *store.Catalog {
	t.Helper()
	start(t)
	ctx := context.Background()
	c, err := store.Open(ctx, pgDSN)
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	if err := c.TruncateAll(ctx); err != nil {
		c.Close()
		t.Fatalf("truncate: %v", err)
	}
	t.Cleanup(c.Close)
	return c
}

func start(t *testing.T) {
	t.Helper()
	pgOnce.Do(func() {
		port, err := freePort()
		if err != nil {
			pgErr = err
			return
		}
		cfg := embeddedpostgres.DefaultConfig().
			Username("ofertas").
			Password("ofertas").
			Database("ofertas").
			Port(uint32(port)).
			RuntimePath(filepath.Join(os.TempDir(), fmt.Sprintf("ofertas-pg-%d", os.Getpid()))).
			StartTimeout(45 * time.Second)
		pgServer = embeddedpostgres.NewDatabase(cfg)
		if err := pgServer.Start(); err != nil {
			pgErr = fmt.Errorf("embedded-postgres start: %w", err)
			return
		}
		pgDSN = fmt.Sprintf("postgres://ofertas:ofertas@127.0.0.1:%d/ofertas?sslmode=disable", port)
	})
	if pgErr != nil {
		t.Fatalf("postgres de teste: %v", pgErr)
	}
}

func Stop() {
	if pgServer != nil {
		_ = pgServer.Stop()
		pgServer = nil
	}
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
