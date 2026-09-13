package domain

type Intencao string

const (
	IntencaoLista    Intencao = "lista"
	IntencaoConsulta Intencao = "consulta"
	IntencaoRecusa   Intencao = "recusa"
)

type Classificacao struct {
	Intencao Intencao
	Texto    string
}

const TextoRecusa = "Só respondo sobre a lista de compras e sobre o Pague Menos Mercado."

const DescricaoPagueMenosMercado = "Pague Menos Mercado é seu assistente virtual para economizar nas compras. Informe sua lista de produtos e descubra onde comprar cada item pelo menor preço, encontrando o mercado mais vantajoso para sua compra. Compare preços, monte sua lista e economize tempo e dinheiro de forma simples, rápida e inteligente."
