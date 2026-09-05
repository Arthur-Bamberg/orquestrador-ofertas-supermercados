package shopfully_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/shopfully"
)

func TestClient_ListagemAndPaginas_FollowsViewerToHighestResJPEG(t *testing.T) {
	listagem, _ := os.ReadFile("testdata/listagem.html")
	viewer, _ := os.ReadFile("testdata/viewer.html")
	pub, _ := os.ReadFile("testdata/publication_stok.json")
	pages, _ := os.ReadFile("testdata/pages_stok.json")
	jpeg := []byte{0xff, 0xd8, 0xff, 0xd9}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/canoas/supermercados":
			w.Header().Set("Content-Type", "text/html")
			w.Write(listagem)
		case strings.Contains(r.URL.RawQuery, "flyerId=1379018"):
			w.Header().Set("Content-Type", "text/html")
			w.Write(viewer)
		case r.URL.Path == "/publication/pt_br_858250":
			w.Header().Set("Content-Type", "application/json")
			w.Write(rewriteHost(pub, r.Host))
		case strings.Contains(r.URL.Path, "/publication_pages/pt_br/858250/"):
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

	c := shopfully.New(srv.Client(), shopfully.Config{
		SiteBase:        srv.URL,
		PublicationBase: srv.URL,
	})

	encartes, err := c.Listagem(context.Background(), "/canoas/supermercados")
	if err != nil {
		t.Fatal(err)
	}
	if len(encartes) != 4 {
		t.Fatalf("listagem=%d", len(encartes))
	}

	paginas, err := c.Paginas(context.Background(), encartes[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(paginas) != 1 || paginas[0].Numero != 1 {
		t.Fatalf("paginas=%+v", paginas)
	}
	if !strings.HasSuffix(paginas[0].URL, "/publications/page_assets/858250/1/page_1_level_5_1147866633.jpeg") {
		t.Fatalf("url=%s", paginas[0].URL)
	}

	body, err := c.Download(context.Background(), paginas[0].URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != string(jpeg) {
		t.Fatalf("bytes=%v", body)
	}
}

func TestClient_Download_RejectsNonImage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html>nope</html>"))
	}))
	defer srv.Close()
	c := shopfully.New(srv.Client(), shopfully.Config{SiteBase: srv.URL})
	_, err := c.Download(context.Background(), srv.URL+"/logo.png")
	if err == nil {
		t.Fatal("expected error")
	}
}

func rewriteHost(raw []byte, host string) []byte {
	dest := "http://" + host
	s := string(raw)
	s = strings.ReplaceAll(s, "shopfully-publication-api.global.ssl.fastly.net", dest)
	s = strings.ReplaceAll(s, "pt-br-media-publications.shopfully.cloud", dest)
	s = strings.ReplaceAll(s, "api-viewer-zmags.shopfully.cloud", dest)
	return []byte(s)
}
