package whatsapp_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/whatsapp"
)

func TestCliente_enviarTextoGraph(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/phone-1/messages" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Fatalf("auth %s", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.out"}]}`))
	}))
	t.Cleanup(srv.Close)
	c := &whatsapp.Cliente{
		HTTP:          srv.Client(),
		BaseURL:       srv.URL,
		Token:         "tok",
		PhoneNumberID: "phone-1",
		DisplayJID:    "5511988887777@s.whatsapp.net",
	}
	if !c.Pronto() || c.Situacao().Estado != domain.CanalPronto {
		t.Fatalf("situacao %+v", c.Situacao())
	}
	id, err := c.Enviar(context.Background(), domain.Envio{
		Destino: "5511999999999@s.whatsapp.net",
		Corpo:   "oi",
	})
	if err != nil || id != "wamid.out" {
		t.Fatalf("id=%s err=%v", id, err)
	}
	if got["to"] != "5511999999999" || got["type"] != "text" {
		t.Fatalf("%+v", got)
	}
}

func TestCliente_enviarTemplate(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.tpl"}]}`))
	}))
	t.Cleanup(srv.Close)
	c := &whatsapp.Cliente{HTTP: srv.Client(), BaseURL: srv.URL, Token: "tok", PhoneNumberID: "phone-1"}
	id, err := c.Enviar(context.Background(), domain.Envio{
		Destino:  "5511999999999",
		Template: &domain.Template{Nome: "hello_world"},
	})
	if err != nil || id != "wamid.tpl" {
		t.Fatalf("id=%s err=%v", id, err)
	}
	if got["type"] != "template" {
		t.Fatalf("%+v", got)
	}
	tpl := got["template"].(map[string]any)
	if tpl["name"] != "hello_world" {
		t.Fatalf("tpl %+v", tpl)
	}
}

func TestCliente_graphForaDaJanela(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":131047,"message":"outside window"}}`))
	}))
	t.Cleanup(srv.Close)
	c := &whatsapp.Cliente{HTTP: srv.Client(), BaseURL: srv.URL, Token: "tok", PhoneNumberID: "phone-1"}
	_, err := c.Enviar(context.Background(), domain.Envio{Destino: "5511999999999", Corpo: "oi"})
	if !errors.Is(err, domain.ErrForaDaJanela) {
		t.Fatalf("err=%v", err)
	}
}

func TestCliente_naoConfigurado(t *testing.T) {
	c := &whatsapp.Cliente{}
	if c.Pronto() || c.Situacao().Estado != domain.CanalNaoConfigurado {
		t.Fatalf("%+v", c.Situacao())
	}
	_, err := c.Enviar(context.Background(), domain.Envio{Destino: "5511", Corpo: "x"})
	if !errors.Is(err, domain.ErrCanalNaoConfigurado) {
		t.Fatalf("err=%v", err)
	}
}

func TestCliente_baixarMidia(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v21.0/media-1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"url":"PLACEHOLDER","mime_type":"image/png"}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mux.HandleFunc("/bin", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Fatalf("auth")
		}
		_, _ = w.Write([]byte("PNG"))
	})
	// rewrite placeholder after we know URL — intercept via custom transport
	c := &whatsapp.Cliente{
		HTTP:          rewriteClient(srv, "/v21.0/media-1", `{"url":"`+srv.URL+"/bin"+`","mime_type":"image/png"}`),
		BaseURL:       srv.URL,
		Token:         "tok",
		PhoneNumberID: "phone-1",
	}
	got, err := c.BaixarMidia(context.Background(), "media-1", domain.MidiaImagem, "", "imagem")
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Conteudo) != "PNG" || got.Tipo != domain.MidiaImagem {
		t.Fatalf("%+v", got)
	}
}

func rewriteClient(srv *httptest.Server, metaPath, metaBody string) *http.Client {
	base := srv.Client()
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == metaPath || strings.HasSuffix(r.URL.Path, metaPath) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(metaBody)),
				Header:     make(http.Header),
				Request:    r,
			}, nil
		}
		return base.Transport.RoundTrip(r)
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTelefone(t *testing.T) {
	if got := whatsapp.Telefone("5511999999999@s.whatsapp.net"); got != "5511999999999" {
		t.Fatalf("%s", got)
	}
}
