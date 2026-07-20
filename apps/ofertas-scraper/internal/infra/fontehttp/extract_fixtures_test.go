package fontehttp

import (
	"net/url"
	"os"
	"testing"
)

func TestExtractFromSavedFixtures(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		base    string
		ct      string
		minPDFs int
		sample  string
	}{
		{"fort", "testdata/fonte-responses/fort/body.html", "https://www.fortatacadista.com.br/ofertas/", "text/html", 10, "MS_Fort_FDS_18-e-19_JUL_26.pdf"},
		{"stok", "testdata/fonte-responses/stok/body.html", "https://stokcenter.com.br/encarte/", "text/html", 1, "55140-Lamina-stok-center-Fim-de-semana-RS-e-SC-18-e-19-de-julho.pdf"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			base, _ := url.Parse(tc.base)
			got := extractPDFLinks(string(body), base, tc.ct)
			t.Logf("%s: %d pdfs", tc.name, len(got))
			if len(got) < tc.minPDFs {
				t.Fatalf("got %d pdfs, want >= %d", len(got), tc.minPDFs)
			}
			found := false
			for _, p := range got {
				if p.Filename == tc.sample {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("missing %s among %d", tc.sample, len(got))
			}
		})
	}
}
