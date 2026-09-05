package application

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/agente-ofertas/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

type Catalogo interface {
	ListarProdutos(ctx context.Context) ([]store.Produto, error)
	ListarMarcas(ctx context.Context) ([]store.Marca, error)
	ListarOfertas(ctx context.Context) ([]store.Oferta, error)
	GetMercado(ctx context.Context, id store.MercadoID) (store.Mercado, bool, error)
}

type Envio interface {
	Enviar(ctx context.Context, conversaJID, corpo string) error
}

type Deps struct {
	Cat   Catalogo
	Envio Envio
	Hoje  func() time.Time
}

type Agente struct {
	cat   Catalogo
	envio Envio
	hoje  func() time.Time
}

func New(d Deps) *Agente {
	hoje := d.Hoje
	if hoje == nil {
		hoje = time.Now
	}
	return &Agente{cat: d.Cat, envio: d.Envio, hoje: hoje}
}

func (a *Agente) Catalogo() Catalogo { return a.cat }

func (a *Agente) Agora() time.Time { return a.hoje() }

func (a *Agente) Atender(ctx context.Context, conversaJID, corpo string) error {
	lista := domain.ParseLista(corpo)
	if len(lista.Itens) == 0 {
		return nil
	}
	texto, err := Interpretar(ctx, lista, a.cat, a.hoje())
	if err != nil {
		return err
	}
	if a.envio == nil || strings.TrimSpace(texto) == "" {
		return nil
	}
	return a.envio.Enviar(ctx, conversaJID, texto)
}

func Interpretar(ctx context.Context, lista domain.Lista, cat Catalogo, agora time.Time) (string, error) {
	produtos, err := cat.ListarProdutos(ctx)
	if err != nil {
		return "", err
	}
	marcas, err := cat.ListarMarcas(ctx)
	if err != nil {
		return "", err
	}
	ofertas, err := cat.ListarOfertas(ctx)
	if err != nil {
		return "", err
	}
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		loc = time.UTC
	}
	hoje := agora.In(loc).Format("2006-01-02")
	blocos := make([]string, 0, len(lista.Itens))
	for _, item := range lista.Itens {
		bloco, err := interpretarItem(ctx, item, produtos, marcas, ofertas, cat, hoje)
		if err != nil {
			return "", err
		}
		blocos = append(blocos, "*"+item.Texto+"*\n"+bloco)
	}
	return strings.Join(blocos, "\n\n"), nil
}

func interpretarItem(ctx context.Context, item domain.Item, produtos []store.Produto, marcas []store.Marca, ofertas []store.Oferta, cat Catalogo, hoje string) (string, error) {
	resto, marca, temMarca := separarMarca(normalizar(item.Texto), marcas)
	achados := casarProdutos(resto, produtos)
	if len(achados) == 0 {
		return "Não encontrei no catálogo.", nil
	}
	if len(achados) > 1 {
		nomes := make([]string, len(achados))
		for i, p := range achados {
			nomes[i] = p.Nome
		}
		sort.Strings(nomes)
		return "Vários produtos: " + strings.Join(nomes, ", ") + ". Manda o nome mais específico.", nil
	}
	p := achados[0]
	var vigentes []store.Oferta
	for _, o := range ofertas {
		if o.ProdutoID != p.ID {
			continue
		}
		if temMarca {
			if o.MarcaID == nil || *o.MarcaID != marca.ID {
				continue
			}
		}
		if o.DataInicio <= hoje && hoje <= o.DataExpiracao {
			vigentes = append(vigentes, o)
		}
	}
	if len(vigentes) == 0 {
		return p.Nome + " — nenhuma Oferta vigente hoje.", nil
	}
	baratas := soMaisBaratas(vigentes)
	type linha struct {
		nome  string
		texto string
	}
	var linhas []linha
	for _, o := range baratas {
		mercado, ok, err := cat.GetMercado(ctx, o.MercadoID)
		if err != nil {
			return "", err
		}
		nome := string(o.MercadoID)
		if ok {
			nome = mercado.Nome
		}
		linhas = append(linhas, linha{nome: nome, texto: formatarOferta(nome, o)})
	}
	sort.Slice(linhas, func(i, j int) bool { return linhas[i].nome < linhas[j].nome })
	out := p.Nome
	for _, l := range linhas {
		out += "\n- " + l.texto
	}
	return out, nil
}

func soMaisBaratas(ofertas []store.Oferta) []store.Oferta {
	if len(ofertas) == 0 {
		return nil
	}
	melhor := precoEfetivo(ofertas[0])
	for _, o := range ofertas[1:] {
		if p := precoEfetivo(o); p < melhor {
			melhor = p
		}
	}
	var out []store.Oferta
	for _, o := range ofertas {
		if precoEfetivo(o) == melhor {
			out = append(out, o)
		}
	}
	return out
}

func precoEfetivo(o store.Oferta) float64 {
	if o.Promocao != nil {
		return o.Promocao.ValorPromocional
	}
	return o.Valor
}

func formatarOferta(mercado string, o store.Oferta) string {
	qtds := make([]string, 0, len(o.Quantidades))
	for _, q := range o.Quantidades {
		qtds = append(qtds, formatarQtd(q, o.Medida))
	}
	s := mercado + " — " + reais(o.Valor) + " / " + strings.Join(qtds, ", ")
	if o.Promocao != nil {
		s += " (promoção " + reais(o.Promocao.ValorPromocional) + ")"
	}
	return s
}

func formatarQtd(q float64, m store.Medida) string {
	n := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", q), "0"), ".")
	return n + " " + string(m)
}

func reais(v float64) string {
	return "R$ " + strings.Replace(fmt.Sprintf("%.2f", v), ".", ",", 1)
}

func normalizar(s string) string {
	fields := strings.Fields(strings.TrimSpace(s))
	if len(fields) == 0 {
		return ""
	}
	return strings.ToLower(strings.Join(fields, " "))
}

func separarMarca(norm string, marcas []store.Marca) (string, store.Marca, bool) {
	best := -1
	var found store.Marca
	resto := norm
	for _, m := range marcas {
		if m.NomeNorm == "" || !contemToken(norm, m.NomeNorm) {
			continue
		}
		if len(m.NomeNorm) > best {
			best = len(m.NomeNorm)
			found = m
			resto = strings.Join(strings.Fields(strings.ReplaceAll(norm, m.NomeNorm, " ")), " ")
		}
	}
	return resto, found, best >= 0
}

func contemToken(hay, needle string) bool {
	return strings.Contains(" "+hay+" ", " "+needle+" ")
}

func casarProdutos(needle string, produtos []store.Produto) []store.Produto {
	if needle == "" {
		return nil
	}
	var exact, fuzzy []store.Produto
	runes := utf8Len(needle)
	for _, p := range produtos {
		if p.NomeNorm == needle {
			exact = append(exact, p)
			continue
		}
		if runes < 3 {
			continue
		}
		if strings.Contains(p.NomeNorm, needle) || (utf8Len(p.NomeNorm) >= 3 && strings.Contains(needle, p.NomeNorm)) {
			fuzzy = append(fuzzy, p)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	return fuzzy
}

func utf8Len(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
