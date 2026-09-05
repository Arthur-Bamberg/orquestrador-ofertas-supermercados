package domain

// Encarte is a flyer listed on Shopfully for one Mercado.
type Encarte struct {
	ID         string
	Mercado    string
	ViewerPath string
	Vencimento string
}

// Pagina is one page image of an Encarte.
type Pagina struct {
	Numero int
	URL    string
}
