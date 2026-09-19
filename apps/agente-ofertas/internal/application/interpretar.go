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

type InterpretadorTermo interface {
	Termo(ctx context.Context, item string) (string, error)
}

type Candidato struct {
	ID          store.OfertaID
	Produto     string
	Marca       string
	Mercado     string
	Valor       float64
	Quantidades []float64
	Medida      store.Medida
}

type Escolha interface {
	Escolher(ctx context.Context, item string, candidatos []Candidato) ([]store.OfertaID, error)
}

type ClassificadorIntencao interface {
	Classificar(ctx context.Context, texto string) (domain.Classificacao, error)
}

type Envio interface {
	Enviar(ctx context.Context, conversaJID, corpo string) error
}

type Deps struct {
	Cat      Catalogo
	Coleta   Coleta
	Termo    InterpretadorTermo
	Escolha  Escolha
	Intencao ClassificadorIntencao
	Envio    Envio
	Hoje     func() time.Time
}

type Agente struct {
	cat      Catalogo
	coleta   Coleta
	termo    InterpretadorTermo
	escolha  Escolha
	intencao ClassificadorIntencao
	envio    Envio
	hoje     func() time.Time
}

func New(d Deps) *Agente {
	hoje := d.Hoje
	if hoje == nil {
		hoje = time.Now
	}
	return &Agente{cat: d.Cat, coleta: d.Coleta, termo: d.Termo, escolha: d.Escolha, intencao: d.Intencao, envio: d.Envio, hoje: hoje}
}

func (a *Agente) Agora() time.Time { return a.hoje() }

func (a *Agente) InterpretarLista(ctx context.Context, texto string) (string, error) {
	return a.textoDaIntencao(ctx, texto, a.classificar(ctx, texto))
}

func (a *Agente) Atender(ctx context.Context, conversaJID, corpo string) error {
	cl := a.classificar(ctx, corpo)
	if (cl.Intencao == domain.IntencaoConsulta || cl.Intencao == domain.IntencaoRecusa) && conversaGrupo(conversaJID) {
		return nil
	}
	texto, err := a.textoDaIntencao(ctx, corpo, cl)
	if err != nil {
		return err
	}
	if a.envio == nil || strings.TrimSpace(texto) == "" {
		return nil
	}
	return a.envio.Enviar(ctx, conversaJID, texto)
}

func (a *Agente) textoDaIntencao(ctx context.Context, texto string, cl domain.Classificacao) (string, error) {
	switch cl.Intencao {
	case domain.IntencaoConsulta:
		if strings.TrimSpace(cl.Texto) == "" {
			return domain.DescricaoPagueMenosMercado, nil
		}
		return cl.Texto, nil
	case domain.IntencaoRecusa:
		return domain.TextoRecusa, nil
	}
	lista := domain.ParseLista(texto)
	if len(lista.Itens) == 0 {
		return "", nil
	}
	return interpretar(ctx, lista, a.cat, a.coleta, a.hoje(), a.termo, a.escolha)
}

func (a *Agente) classificar(ctx context.Context, texto string) domain.Classificacao {
	if a.intencao == nil {
		return domain.Classificacao{Intencao: domain.IntencaoLista}
	}
	cl, err := a.intencao.Classificar(ctx, texto)
	if err != nil {
		return domain.Classificacao{Intencao: domain.IntencaoLista}
	}
	return cl
}

func conversaGrupo(jid string) bool {
	return strings.HasSuffix(jid, "@g.us")
}

func Interpretar(ctx context.Context, lista domain.Lista, cat Catalogo, coleta Coleta, agora time.Time, interp InterpretadorTermo) (string, error) {
	return interpretar(ctx, lista, cat, coleta, agora, interp, nil)
}

func interpretar(ctx context.Context, lista domain.Lista, cat Catalogo, coleta Coleta, agora time.Time, interp InterpretadorTermo, escolha Escolha) (string, error) {
	termos := make([]string, len(lista.Itens))
	for i, item := range lista.Itens {
		termos[i] = resolverTermo(ctx, item.Texto, interp)
	}
	consultados := coletarItens(ctx, termos, coleta)
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
		bloco, err := interpretarItem(ctx, item.Texto, termos[i], produtos, marcas, ofertas, consultados[i], cat, hoje, escolha)
		if err != nil {
			return "", err
		}
		blocos = append(blocos, "*"+item.Texto+"*\n"+bloco)
	}
	return strings.Join(blocos, "\n\n"), nil
}

func resolverTermo(ctx context.Context, item string, interp InterpretadorTermo) string {
	if interp != nil {
		t, err := interp.Termo(ctx, item)
		if err == nil {
			t = strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(t))), " ")
			if t != "" {
				return t
			}
		}
	}
	return domain.TermoDoItem(item)
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

func coletarItens(ctx context.Context, termos []string, coleta Coleta) [][]store.Oferta {
	out := make([][]store.Oferta, len(termos))
	if coleta == nil {
		return out
	}
	var wg sync.WaitGroup
	for i, termo := range termos {
		if termo == "" {
			continue
		}
		wg.Add(1)
		go func(i int, termo string) {
			defer wg.Done()
			of, err := coleta.Coletar(ctx, termo)
			if err != nil {
				return
			}
			out[i] = of
		}(i, termo)
	}
	wg.Wait()
	return out
}

func interpretarItem(ctx context.Context, item, termo string, produtos []store.Produto, marcas []store.Marca, ofertas, consultado []store.Oferta, cat Catalogo, hoje string, escolha Escolha) (string, error) {
	if termo == "" {
		return "Não achei.", nil
	}
	resto, marca, temMarca := separarMarca(normalizar(termo), marcas)
	tipo := domain.TipoDoTermo(resto)
	encarte := encarteCasado(ofertas, produtos, tipo)
	merged := fundirConsultadoEEncarte(consultado, encarte, hoje)
	if temMarca {
		var soMarca []store.Oferta
		for _, o := range merged {
			if o.MarcaID != nil && *o.MarcaID == marca.ID {
				soMarca = append(soMarca, o)
			}
		}
		merged = soMarca
	}
	escolhidas, viaEscolha, err := escolherOfertas(ctx, item, merged, produtos, marcas, cat, escolha)
	if err != nil {
		return "", err
	}
	if viaEscolha {
		merged = escolhidas
	}
	porProduto := map[store.ProdutoID][]store.Oferta{}
	for _, o := range merged {
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
		p, found := prodByID[id]
		if !found {
			continue
		}
		if !viaEscolha && !tipoVale(tipo, p.NomeNorm) {
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
	tipos = escolherProdutos(tipo, tipos, porProduto)
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

func escolherOfertas(ctx context.Context, item string, ofertas []store.Oferta, produtos []store.Produto, marcas []store.Marca, cat Catalogo, escolha Escolha) ([]store.Oferta, bool, error) {
	if escolha == nil || len(ofertas) == 0 {
		return nil, false, nil
	}
	prodByID := map[store.ProdutoID]store.Produto{}
	for _, p := range produtos {
		prodByID[p.ID] = p
	}
	candidatos := make([]Candidato, 0, len(ofertas))
	for _, o := range ofertas {
		p := prodByID[o.ProdutoID]
		mercado, found, err := cat.GetMercado(ctx, o.MercadoID)
		if err != nil {
			return nil, false, err
		}
		nomeMercado := string(o.MercadoID)
		if found {
			nomeMercado = mercado.Nome
		}
		candidatos = append(candidatos, Candidato{
			ID:          o.ID,
			Produto:     p.Nome,
			Marca:       nomeMarca(o, marcas),
			Mercado:     nomeMercado,
			Valor:       precoEfetivo(o),
			Quantidades: o.Quantidades,
			Medida:      o.Medida,
		})
	}
	ids, err := escolha.Escolher(ctx, item, candidatos)
	if err != nil {
		return nil, false, nil
	}
	keep := map[store.OfertaID]struct{}{}
	for _, id := range ids {
		keep[id] = struct{}{}
	}
	var out []store.Oferta
	for _, o := range ofertas {
		if _, ok := keep[o.ID]; ok {
			out = append(out, o)
		}
	}
	return out, true, nil
}

func escolherProdutos(termo string, tipos []store.Produto, porProduto map[store.ProdutoID][]store.Oferta) []store.Produto {
	minEx := extrasAlem(termo, tipos[0].NomeNorm)
	for _, p := range tipos[1:] {
		if e := extrasAlem(termo, p.NomeNorm); e < minEx {
			minEx = e
		}
	}
	var candidatos []store.Produto
	for _, p := range tipos {
		if extrasAlem(termo, p.NomeNorm) == minEx {
			candidatos = append(candidatos, p)
		}
	}
	if len(candidatos) == 1 {
		return candidatos
	}
	minPreco := menorPreco(porProduto[candidatos[0].ID])
	for _, p := range candidatos[1:] {
		if v := menorPreco(porProduto[p.ID]); v < minPreco {
			minPreco = v
		}
	}
	var out []store.Produto
	for _, p := range candidatos {
		if menorPreco(porProduto[p.ID]) == minPreco {
			out = append(out, p)
		}
	}
	return out
}

func extrasAlem(termo, nomeNorm string) int {
	return len(tokensDoTipo(nomeNorm)) - len(tokensDoTipo(termo))
}

func menorPreco(ofertas []store.Oferta) float64 {
	baratas := soMaisBaratas(ofertas)
	if len(baratas) == 0 {
		return 0
	}
	return precoEfetivo(baratas[0])
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
	itemToks := tokensDoTipo(item)
	nomeToks := tokensDoTipo(nomeNorm)
	if len(itemToks) == 0 || len(nomeToks) == 0 {
		return false
	}
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

func tokensDoTipo(s string) []string {
	return domain.TokensSemPreposicao(s)
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
	needleN := tipoComparavel(needle)
	if needleN == "" {
		return nil
	}
	var exact, fuzzy []store.Produto
	runes := utf8Len(needleN)
	for _, p := range produtos {
		nomeN := tipoComparavel(p.NomeNorm)
		if nomeN == needleN {
			exact = append(exact, p)
			continue
		}
		if runes < 3 {
			continue
		}
		if strings.Contains(nomeN, needleN) || (utf8Len(nomeN) >= 3 && strings.Contains(needleN, nomeN)) {
			fuzzy = append(fuzzy, p)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	return fuzzy
}

func tipoComparavel(s string) string {
	return strings.Join(tokensDoTipo(s), " ")
}

func utf8Len(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
