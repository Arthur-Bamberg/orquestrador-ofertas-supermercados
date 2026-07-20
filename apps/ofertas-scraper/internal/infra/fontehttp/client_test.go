package fontehttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/infra/fontehttp"
)

func TestDiscoverPDFs_CollectsAbsoluteAndRelativePDFLinks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body>
			<a href="https://cdn.example/a/ofertas.pdf">a</a>
			<a href="/rel/outro.pdf">b</a>
			<a href="/nope.txt">c</a>
		</body></html>`))
	}))
	defer srv.Close()

	client := fontehttp.New(srv.Client())
	got, err := client.DiscoverPDFs(context.Background(), domain.Fonte{URL: srv.URL + "/page"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	if got[0].Filename != "ofertas.pdf" || got[0].URL != "https://cdn.example/a/ofertas.pdf" {
		t.Fatalf("first=%+v", got[0])
	}
	if got[1].Filename != "outro.pdf" {
		t.Fatalf("second=%+v", got[1])
	}
	if got[1].URL != srv.URL+"/rel/outro.pdf" {
		t.Fatalf("resolved url=%s", got[1].URL)
	}
}

func TestDiscoverPDFs_FindsPDFInJSJSONEscaped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><script>
			var ofertas = [{"foto":"https:\/\/www.fortatacadista.com.br\/wp-content\/uploads\/2026\/07\/MS_Fort_FDS.pdf"}];
			window.option_df = {"source":"https:\/\/stokcenter.com.br\/wp-content\/uploads\/2026\/07\/lamina.pdf"};
		</script></html>`))
	}))
	defer srv.Close()

	got, err := fontehttp.New(srv.Client()).DiscoverPDFs(context.Background(), domain.Fonte{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	want := map[string]string{
		"MS_Fort_FDS.pdf": "https://www.fortatacadista.com.br/wp-content/uploads/2026/07/MS_Fort_FDS.pdf",
		"lamina.pdf":      "https://stokcenter.com.br/wp-content/uploads/2026/07/lamina.pdf",
	}
	for _, p := range got {
		if want[p.Filename] != p.URL {
			t.Fatalf("unexpected %+v", p)
		}
	}
}

func TestDiscoverPDFs_JSONAPIBareArquivo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"nome":"Fim de Semana","arquivo":"Fim-de-Semana_17.pdf"}]`))
	}))
	defer srv.Close()

	// Simulate Via-style /api/ofertas path so bare files resolve under /encarte/
	fonteURL := srv.URL + "/via-atacadista/api/ofertas?loja=abc"
	got, err := fontehttp.New(srv.Client()).DiscoverPDFs(context.Background(), domain.Fonte{URL: fonteURL})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %+v", got)
	}
	want := srv.URL + "/via-atacadista/encarte/Fim-de-Semana_17.pdf"
	if got[0].URL != want || got[0].Filename != "Fim-de-Semana_17.pdf" {
		t.Fatalf("got %+v want url %s", got[0], want)
	}
}

func TestDiscoverPDFs_SavedFortFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/fonte-responses/fort/body.html")
	if err != nil {
		t.Skip(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	got, err := fontehttp.New(srv.Client()).DiscoverPDFs(context.Background(), domain.Fonte{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("expected PDFs from Fort fixture")
	}
	found := false
	for _, p := range got {
		if p.Filename == "MS_Fort_FDS_18-e-19_JUL_26.pdf" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing Fort weekend PDF in %+v", got)
	}
}
