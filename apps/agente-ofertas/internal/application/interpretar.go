package application

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
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

type Coleta interface {
	Coletar(ctx context.Context, termo string) ([]store.Oferta, error)
}

type Envio interface {
	Enviar(ctx context.Context, conversaJID, corpo string) error
}

type Deps struct {
	Cat    Catalogo
	Coleta Coleta
	Envio  Envio
	Hoje   func() time.Time
}

type Agente struct {
	cat    Catalogo
	coleta Coleta
	envio  Envio
	hoje   func() time.Time
}

func New(d Deps) *Agente {
	hoje := d.Hoje
	if hoje == nil {
		hoje = time.Now
	}
	return &Agente{cat: d.Cat, coleta: d.Coleta, envio: d.Envio, hoje: hoje}
}

func (a *Agente) Agora() time.Time { return a.hoje() }

func (a *Agente) InterpretarLista(ctx context.Context, texto string) (string, error) {
	return Interpretar(ctx, domain.ParseLista(texto), a.cat, a.coleta, a.hoje())
}

func (a *Agente) Atender(ctx context.Context, conversaJID, corpo string) error {
	lista := domain.ParseLista(corpo)
	if len(lista.Itens) == 0 {
		return nil
	}
	texto, err := Interpretar(ctx, lista, a.cat, a.coleta, a.hoje())
	if err != nil {
		return err
	}
	if a.envio == nil || strings.TrimSpace(texto) == "" {
		return nil
	}
	return a.envio.Enviar(ctx, conversaJID, texto)
}

func Interpretar(ctx context.Context, lista domain.Lista, cat Catalogo, coleta Coleta, agora time.Time) (string, error) {
	consultados := coletarItens(ctx, lista, coleta)
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
	for i, item := range lista.Itens {
		bloco, err := interpretarItem(ctx, item, produtos, marcas, ofertas, consultados[i], cat, hoje)
		if err != nil {
			return "", err
		}
		blocos = append(blocos, "*"+item.Texto+"*\n"+bloco)
	}
	return strings.Join(blocos, "\n\n"), nil
}

func fundirConsultadoEEncarte(consultado, encarte []store.Oferta, hoje string) []store.Oferta {
	type chave struct {
		p store.ProdutoID
		m store.MercadoID
	}
	trazido := map[chave]struct{}{}
	var out []store.Oferta
	for _, o := range consultado {
		if !vigenteHoje(o, hoje) {
			continue
		}
		trazido[chave{o.ProdutoID, o.MercadoID}] = struct{}{}
		out = append(out, o)
	}
	var soEncarte []store.Oferta
	for _, o := range encarte {
		if o.DocumentoID == "" {
			continue
		}
		soEncarte = append(soEncarte, o)
	}
	for _, o := range soEncarte {
		if !vigenteHoje(o, hoje) {
			continue
		}
		if _, ok := trazido[chave{o.ProdutoID, o.MercadoID}]; ok {
			continue
		}
		out = append(out, o)
	}
	return out
}

func vigenteHoje(o store.Oferta, hoje string) bool {
	return o.DataInicio <= hoje && hoje <= o.DataExpiracao
}

func coletarItens(ctx context.Context, lista domain.Lista, coleta Coleta) [][]store.Oferta {
	out := make([][]store.Oferta, len(lista.Itens))
	if coleta == nil {
		return out
	}
	var wg sync.WaitGroup
	for i, item := range lista.Itens {
		wg.Add(1)
		go func(i int, termo string) {
			defer wg.Done()
			of, err := coleta.Coletar(ctx, termo)
			if err != nil {
				return
			}
			out[i] = of
		}(i, item.Texto)
	}
	wg.Wait()
	return out
}

func interpretarItem(ctx context.Context, item domain.Item, produtos []store.Produto, marcas []store.Marca, ofertas, consultado []store.Oferta, cat Catalogo, hoje string) (string, error) {
	resto, marca, temMarca := separarMarca(normalizar(item.Texto), marcas)
	encarte := encarteCasado(ofertas, produtos, resto)
	merged := fundirConsultadoEEncarte(consultado, encarte, hoje)
	porProduto := map[store.ProdutoID][]store.Oferta{}
	for _, o := range merged {
		if temMarca {
			if o.MarcaID == nil || *o.MarcaID != marca.ID {
				continue
			}
		}
		porProduto[o.ProdutoID] = append(porProduto[o.ProdutoID], o)
	}
	prodByID := map[store.ProdutoID]store.Produto{}
	for _, p := range produtos {
		prodByID[p.ID] = p
	}
	var tipos []store.Produto
	seen := map[store.ProdutoID]struct{}{}
	for id, ofs := range porProduto {
		if len(ofs) == 0 {
			continue
		}
		p, ok := prodByID[id]
		if !ok || !tipoVale(resto, p.NomeNorm) {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		tipos = append(tipos, p)
	}
	if len(tipos) == 0 {
		return "Não achei.", nil
	}
	sort.Slice(tipos, func(i, j int) bool { return tipos[i].Nome < tipos[j].Nome })
	var blocos []string
	for _, p := range tipos {
		baratas := soMaisBaratas(porProduto[p.ID])
		bloco := p.Nome
		type linha struct{ nome, texto string }
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
			linhas = append(linhas, linha{nome: nome, texto: formatarOferta(nome, o, marcas)})
		}
		sort.Slice(linhas, func(i, j int) bool { return linhas[i].nome < linhas[j].nome })
		for _, l := range linhas {
			bloco += "\n- " + l.texto
		}
		blocos = append(blocos, bloco)
	}
	return strings.Join(blocos, "\n\n"), nil
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

func formatarOferta(mercado string, o store.Oferta, marcas []store.Marca) string {
	qtds := make([]string, 0, len(o.Quantidades))
	for _, q := range o.Quantidades {
		qtds = append(qtds, formatarQtd(q, o.Medida))
	}
	s := mercado
	if m := nomeMarca(o, marcas); m != "" {
		s += " — " + m
	}
	s += " — " + reais(o.Valor) + " / " + strings.Join(qtds, ", ")
	if o.Promocao != nil {
		s += " (promoção " + reais(o.Promocao.ValorPromocional) + ")"
	}
	return s
}

func nomeMarca(o store.Oferta, marcas []store.Marca) string {
	if o.MarcaID == nil {
		return ""
	}
	for _, m := range marcas {
		if m.ID == *o.MarcaID {
			return m.Nome
		}
	}
	return ""
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

func tipoVale(item, nomeNorm string) bool {
	if item == "" || nomeNorm == "" {
		return false
	}
	if nomeNorm == item {
		return true
	}
	itemToks := strings.Fields(item)
	nomeToks := strings.Fields(nomeNorm)
	if len(nomeToks) < len(itemToks) {
		return false
	}
	for i, t := range itemToks {
		if nomeToks[i] != t {
			return false
		}
	}
	return true
}

func encarteCasado(ofertas []store.Oferta, produtos []store.Produto, resto string) []store.Oferta {
	achados := casarProdutos(resto, produtos)
	ids := map[store.ProdutoID]struct{}{}
	for _, p := range achados {
		ids[p.ID] = struct{}{}
	}
	var out []store.Oferta
	for _, o := range ofertas {
		if _, ok := ids[o.ProdutoID]; !ok {
			continue
		}
		out = append(out, o)
	}
	return out
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
