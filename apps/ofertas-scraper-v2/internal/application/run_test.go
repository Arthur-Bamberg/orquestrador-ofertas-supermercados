package application_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/artefato"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/shopfully"
)

func TestRun_DownloadsOnlyPageImagesOfMercadosWeHave(t *testing.T) {
	listagem, err := os.ReadFile("../infra/shopfully/testdata/listagem.html")
	if err != nil {
		t.Fatal(err)
	}
	viewer, err := os.ReadFile("../infra/shopfully/testdata/viewer.html")
	if err != nil {
		t.Fatal(err)
	}
	pub, err := os.ReadFile("../infra/shopfully/testdata/publication_stok.json")
	if err != nil {
		t.Fatal(err)
	}
	pages, err := os.ReadFile("../infra/shopfully/testdata/pages_stok.json")
	if err != nil {
		t.Fatal(err)
	}
	jpeg := []byte{0xff, 0xd8, 0xff, 0xd9}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/canoas/supermercados":
			w.Write(listagem)
		case strings.Contains(r.URL.RawQuery, "flyerId=1379018"):
			w.Write(viewer)
		case strings.Contains(r.URL.RawQuery, "flyerId=1379017"):
			w.Write(viewer)
		case strings.HasPrefix(r.URL.Path, "/publication/"):
			w.Header().Set("Content-Type", "application/json")
			w.Write(rewriteHost(pub, r.Host))
		case strings.Contains(r.URL.Path, "/publication_pages/"):
			w.Header().Set("Content-Type", "application/json")
			w.Write(rewriteHost(pages, r.Host))
		case strings.HasSuffix(r.URL.Path, ".jpeg"):
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write(jpeg)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	root := t.TempDir()
	job := application.Job{
		ListagemPath: "/canoas/supermercados",
		Mercados:     []string{"Fort Atacadista", "Stok Center", "Via Atacadista"},
		Client: shopfully.New(srv.Client(), shopfully.Config{
			SiteBase:        srv.URL,
			PublicationBase: srv.URL,
		}),
		Store: artefato.New(root),
	}

	got, err := job.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Listados != 4 {
		t.Fatalf("listados=%d", got.Listados)
	}
	if got.Filtrados != 2 {
		t.Fatalf("filtrados=%d (só Stok Center)", got.Filtrados)
	}
	if got.Baixados != 2 {
		t.Fatalf("baixados=%d %+v", got.Baixados, got.Arquivos)
	}
	for _, p := range got.Arquivos {
		if !strings.Contains(p, "stok-center") {
			t.Fatalf("arquivo de mercado inesperado: %s", p)
		}
		if !strings.Contains(p, "pagina-") {
			t.Fatalf("nao e pagina do encarte: %s", p)
		}
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Fatalf("missing %s: %v", p, err)
		}
	}
}

func TestRun_SkipsEncarteWhenViewerFailsAndKeepsTheOther(t *testing.T) {
	listagem, err := os.ReadFile("../infra/shopfully/testdata/listagem.html")
	if err != nil {
		t.Fatal(err)
	}
	viewer, err := os.ReadFile("../infra/shopfully/testdata/viewer.html")
	if err != nil {
		t.Fatal(err)
	}
	pub, err := os.ReadFile("../infra/shopfully/testdata/publication_stok.json")
	if err != nil {
		t.Fatal(err)
	}
	pages, err := os.ReadFile("../infra/shopfully/testdata/pages_stok.json")
	if err != nil {
		t.Fatal(err)
	}
	jpeg := []byte{0xff, 0xd8, 0xff, 0xd9}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/canoas/supermercados":
			w.Write(listagem)
		case strings.Contains(r.URL.RawQuery, "flyerId=1379017"):
			http.Error(w, "gone", http.StatusNotFound)
		case strings.Contains(r.URL.RawQuery, "flyerId=1379018"):
			w.Write(viewer)
		case strings.HasPrefix(r.URL.Path, "/publication/"):
			w.Header().Set("Content-Type", "application/json")
			w.Write(rewriteHost(pub, r.Host))
		case strings.Contains(r.URL.Path, "/publication_pages/"):
			w.Header().Set("Content-Type", "application/json")
			w.Write(rewriteHost(pages, r.Host))
		case strings.HasSuffix(r.URL.Path, ".jpeg"):
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write(jpeg)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	job := application.Job{
		ListagemPath: "/canoas/supermercados",
		Mercados:     []string{"Stok Center"},
		Client: shopfully.New(srv.Client(), shopfully.Config{
			SiteBase:        srv.URL,
			PublicationBase: srv.URL,
		}),
		Store: artefato.New(t.TempDir()),
	}
	got, err := job.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Baixados != 1 {
		t.Fatalf("baixados=%d arquivos=%v", got.Baixados, got.Arquivos)
	}
	if got.Falhas != 1 {
		t.Fatalf("falhas=%d", got.Falhas)
	}
}

func rewriteHost(raw []byte, host string) []byte {
	dest := "http://" + host
	s := string(raw)
	s = strings.ReplaceAll(s, "shopfully-publication-api.global.ssl.fastly.net", dest)
	s = strings.ReplaceAll(s, "pt-br-media-publications.shopfully.cloud", dest)
	return []byte(s)
}
