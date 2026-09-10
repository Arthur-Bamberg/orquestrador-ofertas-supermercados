package application_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestInterpretar_disparaColetaComTermoDeCadaItem(t *testing.T) {
	c := &stubColeta{}
	_, err := application.Interpretar(context.Background(), domain.ParseLista("leite, tomate"), catalogoLeite(), c, dia())
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, t := range c.termos {
		seen[t]++
	}
	if seen["leite"] != 1 || seen["tomate"] != 1 || len(c.termos) != 2 {
		t.Fatalf("termos=%q", c.termos)
	}
}

func TestInterpretar_coletaDoDiaEntraNaResposta(t *testing.T) {
	c := &stubColeta{porTermo: map[string][]store.Oferta{
		"leite": {{
			ID: "c1", ProdutoID: "leite", MercadoID: "stok",
			Valor: 5.5, Quantidades: []float64{1000}, Medida: store.MedidaML,
			DataInicio: "2026-09-03", DataExpiracao: "2026-09-03",
		}},
	}}
	cat := catalogoLeite()
	cat.ofertas = nil
	got, err := application.Interpretar(context.Background(), domain.ParseLista("leite"), cat, c, dia())
	if err != nil {
		t.Fatal(err)
	}
	want := "*leite*\nLeite integral\n- Stok — R$ 5,50 / 1000 ml"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInterpretar_coletaDoDiaPrevaleceNoMesmoProdutoEMercado(t *testing.T) {
	c := &stubColeta{porTermo: map[string][]store.Oferta{
		"leite": {{
			ID: "c1", ProdutoID: "leite", MercadoID: "fort",
			Valor: 5.49, Quantidades: []float64{1000}, Medida: store.MedidaML,
			DataInicio: "2026-09-03", DataExpiracao: "2026-09-03",
		}},
	}}
	cat := catalogoLeite()
	cat.ofertas = []store.Oferta{{
		ID: "e1", ProdutoID: "leite", MercadoID: "fort", DocumentoID: "d1",
		Valor: 3.99, Quantidades: []float64{1000}, Medida: store.MedidaML,
		DataInicio: "2026-09-01", DataExpiracao: "2026-09-10",
	}}
	got, err := application.Interpretar(context.Background(), domain.ParseLista("leite"), cat, c, dia())
	if err != nil {
		t.Fatal(err)
	}
	want := "*leite*\nLeite integral\n- Fort — R$ 5,49 / 1000 ml"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInterpretar_encarteCompletaMercadoSemColetaDoProduto(t *testing.T) {
	c := &stubColeta{porTermo: map[string][]store.Oferta{
		"leite": {{
			ID: "c1", ProdutoID: "leite", MercadoID: "fort",
			Valor: 5.5, Quantidades: []float64{1000}, Medida: store.MedidaML,
			DataInicio: "2026-09-03", DataExpiracao: "2026-09-03",
		}},
	}}
	cat := catalogoLeite()
	cat.ofertas = []store.Oferta{{
		ID: "e1", ProdutoID: "leite", MercadoID: "carrefour", DocumentoID: "d1",
		Valor: 4, Quantidades: []float64{1000}, Medida: store.MedidaML,
		DataInicio: "2026-09-01", DataExpiracao: "2026-09-10",
	}}
	cat.mercados["carrefour"] = store.Mercado{ID: "carrefour", Nome: "Carrefour"}
	got, err := application.Interpretar(context.Background(), domain.ParseLista("leite"), cat, c, dia())
	if err != nil {
		t.Fatal(err)
	}
	want := "*leite*\nLeite integral\n- Carrefour — R$ 4,00 / 1000 ml"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInterpretar_naoUsaOfertaSoDeColetaAntiga(t *testing.T) {
	cat := catalogoLeite()
	cat.ofertas = []store.Oferta{
		{
			ID: "old", ProdutoID: "leite", MercadoID: "stok",
			Valor: 1, Quantidades: []float64{1000}, Medida: store.MedidaML,
			DataInicio: "2026-09-01", DataExpiracao: "2026-09-10",
		},
		{
			ID: "e1", ProdutoID: "leite", MercadoID: "fort", DocumentoID: "d1",
			Valor: 5.9, Quantidades: []float64{1000}, Medida: store.MedidaML,
			DataInicio: "2026-09-01", DataExpiracao: "2026-09-10",
		},
	}
	got, err := application.Interpretar(context.Background(), domain.ParseLista("leite"), cat, coletaVazia(), dia())
	if err != nil {
		t.Fatal(err)
	}
	want := "*leite*\nLeite integral\n- Fort — R$ 5,90 / 1000 ml"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInterpretar_descartaRelacionado(t *testing.T) {
	c := &stubColeta{porTermo: map[string][]store.Oferta{
		"tomate": {
			{
				ID: "c1", ProdutoID: "tomate", MercadoID: "fort",
				Valor: 5.49, Quantidades: []float64{1000}, Medida: store.MedidaG,
				DataInicio: "2026-09-03", DataExpiracao: "2026-09-03",
			},
			{
				ID: "c2", ProdutoID: "molho", MercadoID: "fort",
				Valor: 2, Quantidades: []float64{340}, Medida: store.MedidaG,
				DataInicio: "2026-09-03", DataExpiracao: "2026-09-03",
			},
		},
	}}
	cat := memCat{
		produtos: []store.Produto{
			{ID: "tomate", Nome: "Tomate", NomeNorm: "tomate"},
			{ID: "molho", Nome: "Molho de tomate", NomeNorm: "molho de tomate"},
		},
		mercados: map[store.MercadoID]store.Mercado{"fort": {ID: "fort", Nome: "Fort"}},
	}
	got, err := application.Interpretar(context.Background(), domain.ParseLista("tomate"), cat, c, dia())
	if err != nil {
		t.Fatal(err)
	}
	want := "*tomate*\nTomate\n- Fort — R$ 5,49 / 1000 g"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInterpretar_variosTiposQueValemSubBlocoPorTipo(t *testing.T) {
	c := &stubColeta{porTermo: map[string][]store.Oferta{
		"tomate": {{
			ID: "c1", ProdutoID: "italiano", MercadoID: "asun",
			Valor: 4, Quantidades: []float64{500}, Medida: store.MedidaG,
			DataInicio: "2026-09-03", DataExpiracao: "2026-09-03",
		}},
	}}
	cat := memCat{
		produtos: []store.Produto{
			{ID: "tomate", Nome: "Tomate", NomeNorm: "tomate"},
			{ID: "italiano", Nome: "Tomate italiano", NomeNorm: "tomate italiano"},
		},
		ofertas: []store.Oferta{{
			ID: "e1", ProdutoID: "tomate", MercadoID: "fort", DocumentoID: "d1",
			Valor: 3.99, Quantidades: []float64{1000}, Medida: store.MedidaG,
			DataInicio: "2026-09-01", DataExpiracao: "2026-09-10",
		}},
		mercados: map[store.MercadoID]store.Mercado{
			"fort": {ID: "fort", Nome: "Fort"},
			"asun": {ID: "asun", Nome: "Asun"},
		},
	}
	got, err := application.Interpretar(context.Background(), domain.ParseLista("tomate"), cat, c, dia())
	if err != nil {
		t.Fatal(err)
	}
	want := "*tomate*\nTomate\n- Fort — R$ 3,99 / 1000 g\n\nTomate italiano\n- Asun — R$ 4,00 / 500 g"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInterpretar_coletaFalhouAindaUsaEncarte(t *testing.T) {
	c := &stubColeta{err: context.DeadlineExceeded}
	got, err := application.Interpretar(context.Background(), domain.ParseLista("leite"), catalogoLeite(), c, dia())
	if err != nil {
		t.Fatal(err)
	}
	want := "*leite*\nLeite integral\n- Stok — R$ 5,50 / 1000 ml"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInterpretar_itemSemProduto(t *testing.T) {
	got, err := application.Interpretar(context.Background(), domain.ParseLista("xyzabc"), catalogoVazio(), coletaVazia(), dia())
	if err != nil {
		t.Fatal(err)
	}
	if got != "*xyzabc*\nNão achei." {
		t.Fatalf("%q", got)
	}
}

func TestInterpretar_variosProdutosListaCandidatos(t *testing.T) {
	got, err := application.Interpretar(context.Background(), domain.ParseLista("arroz"), catalogoArroz(), coletaVazia(), dia())
	if err != nil {
		t.Fatal(err)
	}
	if got != "*arroz*\nNão achei." {
		t.Fatalf("%q", got)
	}
}

func TestInterpretar_umProdutoListaSoMercadoMaisBarato(t *testing.T) {
	got, err := application.Interpretar(context.Background(), domain.ParseLista("leite"), catalogoLeite(), coletaVazia(), dia())
	if err != nil {
		t.Fatal(err)
	}
	want := "*leite*\nLeite integral\n- Stok — R$ 5,50 / 1000 ml"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInterpretar_empateNoMenorPrecoListaTodosOsMercados(t *testing.T) {
	got, err := application.Interpretar(context.Background(), domain.ParseLista("leite"), catalogoLeiteEmpate(), coletaVazia(), dia())
	if err != nil {
		t.Fatal(err)
	}
	want := "*leite*\nLeite integral\n- Fort — Italac — R$ 5,50 / 1000 ml\n- Stok — R$ 5,50 / 1000 ml"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInterpretar_variosItensRespostaEmLista(t *testing.T) {
	got, err := application.Interpretar(context.Background(), domain.ParseLista("leite, café"), catalogoLeiteECafe(), coletaVazia(), dia())
	if err != nil {
		t.Fatal(err)
	}
	want := "*leite*\nLeite integral\n- Stok — R$ 5,50 / 1000 ml\n\n*café*\nCafé\n- Fort — R$ 10,00 / 500 g"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInterpretar_marcaNoItemFiltraOfertas(t *testing.T) {
	got, err := application.Interpretar(context.Background(), domain.ParseLista("leite Italac"), catalogoLeite(), coletaVazia(), dia())
	if err != nil {
		t.Fatal(err)
	}
	want := "*leite Italac*\nLeite integral\n- Fort — Italac — R$ 5,90 / 1000 ml"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInterpretar_marcaSemOfertaVigenteNaoCompletaComOutras(t *testing.T) {
	got, err := application.Interpretar(context.Background(), domain.ParseLista("leite Piracanjuba"), catalogoLeite(), coletaVazia(), dia())
	if err != nil {
		t.Fatal(err)
	}
	if got != "*leite Piracanjuba*\nNão achei." {
		t.Fatalf("%q", got)
	}
}

func TestInterpretar_ofertaExpiradaFicaDeFora(t *testing.T) {
	got, err := application.Interpretar(context.Background(), domain.ParseLista("café"), catalogoCafeExpirado(), coletaVazia(), dia())
	if err != nil {
		t.Fatal(err)
	}
	if got != "*café*\nNão achei." {
		t.Fatalf("%q", got)
	}
}

func TestAtender_enviaRespostaNaConversa(t *testing.T) {
	envio := &stubEnvio{}
	ag := application.New(application.Deps{
		Cat:    catalogoLeite(),
		Coleta: coletaVazia(),
		Envio:  envio,
		Hoje:   func() time.Time { return dia() },
	})
	if err := ag.Atender(context.Background(), "5511999999999", "leite"); err != nil {
		t.Fatal(err)
	}
	if len(envio.calls) != 1 || envio.calls[0].jid != "5511999999999" {
		t.Fatalf("%+v", envio.calls)
	}
	if envio.calls[0].corpo != "*leite*\nLeite integral\n- Stok — R$ 5,50 / 1000 ml" {
		t.Fatalf("%q", envio.calls[0].corpo)
	}
}

func TestAtender_listaVaziaNaoEnvia(t *testing.T) {
	envio := &stubEnvio{}
	ag := application.New(application.Deps{Cat: catalogoVazio(), Coleta: coletaVazia(), Envio: envio, Hoje: diaFn})
	if err := ag.Atender(context.Background(), "5511999999999", "   "); err != nil {
		t.Fatal(err)
	}
	if len(envio.calls) != 0 {
		t.Fatalf("%+v", envio.calls)
	}
}

func dia() time.Time {
	return time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
}

func diaFn() time.Time { return dia() }

type stubColeta struct {
	mu       sync.Mutex
	termos   []string
	porTermo map[string][]store.Oferta
	err      error
}

func (s *stubColeta) Coletar(_ context.Context, termo string) ([]store.Oferta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.termos = append(s.termos, termo)
	if s.err != nil {
		return nil, s.err
	}
	if s.porTermo == nil {
		return nil, nil
	}
	return s.porTermo[termo], nil
}

func coletaVazia() *stubColeta { return &stubColeta{} }

type memCat struct {
	produtos []store.Produto
	marcas   []store.Marca
	ofertas  []store.Oferta
	mercados map[store.MercadoID]store.Mercado
}

func (m memCat) ListarProdutos(context.Context) ([]store.Produto, error) { return m.produtos, nil }
func (m memCat) ListarMarcas(context.Context) ([]store.Marca, error)     { return m.marcas, nil }
func (m memCat) ListarOfertas(context.Context) ([]store.Oferta, error)   { return m.ofertas, nil }
func (m memCat) GetMercado(_ context.Context, id store.MercadoID) (store.Mercado, bool, error) {
	x, ok := m.mercados[id]
	return x, ok, nil
}

type stubEnvio struct {
	calls []struct{ jid, corpo string }
}

func (s *stubEnvio) Enviar(_ context.Context, conversaJID, corpo string) error {
	s.calls = append(s.calls, struct{ jid, corpo string }{conversaJID, corpo})
	return nil
}

func catalogoVazio() memCat {
	return memCat{mercados: map[store.MercadoID]store.Mercado{}}
}

func catalogoArroz() memCat {
	return memCat{
		produtos: []store.Produto{
			{ID: "p1", Nome: "Arroz branco", NomeNorm: "arroz branco"},
			{ID: "p2", Nome: "Arroz integral", NomeNorm: "arroz integral"},
		},
		mercados: map[store.MercadoID]store.Mercado{},
	}
}

func catalogoLeite() memCat {
	italac := store.MarcaID("italac")
	return memCat{
		produtos: []store.Produto{{ID: "leite", Nome: "Leite integral", NomeNorm: "leite integral"}},
		marcas: []store.Marca{
			{ID: "italac", Nome: "Italac", NomeNorm: "italac"},
			{ID: "pira", Nome: "Piracanjuba", NomeNorm: "piracanjuba"},
		},
		ofertas: []store.Oferta{
			{
				ID: "o1", ProdutoID: "leite", MarcaID: &italac, MercadoID: "fort", DocumentoID: "d1",
				Valor: 5.9, Quantidades: []float64{1000}, Medida: store.MedidaML,
				DataInicio: "2026-09-01", DataExpiracao: "2026-09-10",
			},
			{
				ID: "o2", ProdutoID: "leite", MercadoID: "stok", DocumentoID: "d1",
				Valor: 5.5, Quantidades: []float64{1000}, Medida: store.MedidaML,
				DataInicio: "2026-09-01", DataExpiracao: "2026-09-10",
			},
		},
		mercados: map[store.MercadoID]store.Mercado{
			"fort": {ID: "fort", Nome: "Fort"},
			"stok": {ID: "stok", Nome: "Stok"},
		},
	}
}

func catalogoLeiteEmpate() memCat {
	c := catalogoLeite()
	c.ofertas[0].Valor = 5.5
	return c
}

func catalogoLeiteECafe() memCat {
	c := catalogoLeite()
	c.produtos = append(c.produtos, store.Produto{ID: "cafe", Nome: "Café", NomeNorm: "café"})
	c.ofertas = append(c.ofertas, store.Oferta{
		ID: "o3", ProdutoID: "cafe", MercadoID: "fort", DocumentoID: "d1",
		Valor: 10, Quantidades: []float64{500}, Medida: store.MedidaG,
		DataInicio: "2026-09-01", DataExpiracao: "2026-09-10",
	})
	return c
}

func catalogoCafeExpirado() memCat {
	return memCat{
		produtos: []store.Produto{{ID: "cafe", Nome: "Café", NomeNorm: "café"}},
		ofertas: []store.Oferta{{
			ID: "ox", ProdutoID: "cafe", MercadoID: "fort", DocumentoID: "d1",
			Valor: 10, Quantidades: []float64{500}, Medida: store.MedidaG,
			DataInicio: "2026-01-01", DataExpiracao: "2026-01-31",
		}},
		mercados: map[store.MercadoID]store.Mercado{"fort": {ID: "fort", Nome: "Fort"}},
	}
}
