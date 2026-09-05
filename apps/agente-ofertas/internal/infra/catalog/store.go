package catalog

import (
	"context"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

type Store struct {
	C *store.Catalog
}

func (s Store) ListarProdutos(ctx context.Context) ([]store.Produto, error) {
	return s.C.ListProdutos(ctx)
}

func (s Store) ListarMarcas(ctx context.Context) ([]store.Marca, error) {
	return s.C.ListMarcas(ctx)
}

func (s Store) ListarOfertas(ctx context.Context) ([]store.Oferta, error) {
	return s.C.ListOfertas(ctx)
}

func (s Store) GetMercado(ctx context.Context, id store.MercadoID) (store.Mercado, bool, error) {
	return s.C.GetMercado(ctx, id)
}
