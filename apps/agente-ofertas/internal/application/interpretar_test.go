package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func TestInterpretar_itemSemProduto(t *testing.T) {
	got, err := application.Interpretar(context.Background(), domain.ParseLista("xyzabc"), catalogoVazio(), dia())
	if err != nil {
		t.Fatal(err)
	}
	if got != "*xyzabc*\nNão encontrei no catálogo." {
		t.Fatalf("%q", got)
	}
}

func TestInterpretar_variosProdutosListaCandidatos(t *testing.T) {
	got, err := application.Interpretar(context.Background(), domain.ParseLista("arroz"), catalogoArroz(), dia())
	if err != nil {
		t.Fatal(err)
	}
	if got != "*arroz*\nVários produtos: Arroz branco, Arroz integral. Manda o nome mais específico." {
		t.Fatalf("%q", got)
	}
}

func TestInterpretar_umProdutoListaSoMercadoMaisBarato(t *testing.T) {
	got, err := application.Interpretar(context.Background(), domain.ParseLista("leite"), catalogoLeite(), dia())
	if err != nil {
		t.Fatal(err)
	}
	want := "*leite*\nLeite integral\n- Stok — R$ 5,50 / 1000 ml"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInterpretar_empateNoMenorPrecoListaTodosOsMercados(t *testing.T) {
	got, err := application.Interpretar(context.Background(), domain.ParseLista("leite"), catalogoLeiteEmpate(), dia())
	if err != nil {
		t.Fatal(err)
	}
	want := "*leite*\nLeite integral\n- Fort — R$ 5,50 / 1000 ml\n- Stok — R$ 5,50 / 1000 ml"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInterpretar_variosItensRespostaEmLista(t *testing.T) {
	got, err := application.Interpretar(context.Background(), domain.ParseLista("leite, café"), catalogoLeiteECafe(), dia())
	if err != nil {
		t.Fatal(err)
	}
	want := "*leite*\nLeite integral\n- Stok — R$ 5,50 / 1000 ml\n\n*café*\nCafé\n- Fort — R$ 10,00 / 500 g"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInterpretar_marcaNoItemFiltraOfertas(t *testing.T) {
	got, err := application.Interpretar(context.Background(), domain.ParseLista("leite Italac"), catalogoLeite(), dia())
	if err != nil {
		t.Fatal(err)
	}
	want := "*leite Italac*\nLeite integral\n- Fort — R$ 5,90 / 1000 ml"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInterpretar_marcaSemOfertaVigenteNaoCompletaComOutras(t *testing.T) {
	got, err := application.Interpretar(context.Background(), domain.ParseLista("leite Piracanjuba"), catalogoLeite(), dia())
	if err != nil {
		t.Fatal(err)
	}
	if got != "*leite Piracanjuba*\nLeite integral — nenhuma Oferta vigente hoje." {
		t.Fatalf("%q", got)
	}
}

func TestInterpretar_ofertaExpiradaFicaDeFora(t *testing.T) {
	got, err := application.Interpretar(context.Background(), domain.ParseLista("café"), catalogoCafeExpirado(), dia())
	if err != nil {
		t.Fatal(err)
	}
	if got != "*café*\nCafé — nenhuma Oferta vigente hoje." {
		t.Fatalf("%q", got)
	}
}

func TestAtender_enviaRespostaNaConversa(t *testing.T) {
	envio := &stubEnvio{}
	ag := application.New(application.Deps{
		Cat:   catalogoLeite(),
		Envio: envio,
		Hoje:  func() time.Time { return dia() },
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
	ag := application.New(application.Deps{Cat: catalogoVazio(), Envio: envio, Hoje: diaFn})
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
				ID: "o1", ProdutoID: "leite", MarcaID: &italac, MercadoID: "fort",
				Valor: 5.9, Quantidades: []float64{1000}, Medida: store.MedidaML,
				DataInicio: "2026-09-01", DataExpiracao: "2026-09-10",
			},
			{
				ID: "o2", ProdutoID: "leite", MercadoID: "stok",
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
		ID: "o3", ProdutoID: "cafe", MercadoID: "fort",
		Valor: 10, Quantidades: []float64{500}, Medida: store.MedidaG,
		DataInicio: "2026-09-01", DataExpiracao: "2026-09-10",
	})
	return c
}

func catalogoCafeExpirado() memCat {
	return memCat{
		produtos: []store.Produto{{ID: "cafe", Nome: "Café", NomeNorm: "café"}},
		ofertas: []store.Oferta{{
			ID: "ox", ProdutoID: "cafe", MercadoID: "fort",
			Valor: 10, Quantidades: []float64{500}, Medida: store.MedidaG,
			DataInicio: "2026-01-01", DataExpiracao: "2026-01-31",
		}},
		mercados: map[store.MercadoID]store.Mercado{"fort": {ID: "fort", Nome: "Fort"}},
	}
}
