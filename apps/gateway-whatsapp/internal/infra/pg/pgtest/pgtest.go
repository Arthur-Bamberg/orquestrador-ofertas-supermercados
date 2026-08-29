package pgtest

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

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/pg"
)

var (
	pgOnce   sync.Once
	pgErr    error
	pgDSN    string
	pgServer *embeddedpostgres.EmbeddedPostgres
)

func New(t *testing.T) *pg.Store {
	t.Helper()
	start(t)
	ctx := context.Background()
	s, err := pg.Open(ctx, pgDSN)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := s.TruncateAll(ctx); err != nil {
		s.Close()
		t.Fatalf("truncate: %v", err)
	}
	t.Cleanup(s.Close)
	return s
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
			RuntimePath(filepath.Join(os.TempDir(), fmt.Sprintf("gw-pg-%d", os.Getpid()))).
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
