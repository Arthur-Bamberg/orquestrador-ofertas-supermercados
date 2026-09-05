package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func newID() string { return uuid.NewString() }

// ColetarProduto runs one Coleta per registered Vitrine for the given Produto and day.
func ColetarProduto(ctx context.Context, catalog *store.Catalog, vitrines map[store.MercadoID]domain.Vitrine, produtoID store.ProdutoID, dia string) error {
	produto, ok, err := catalog.GetProduto(ctx, produtoID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: produto %s", store.ErrNotFound, produtoID)
	}
	marcas := store.NewMarcaRepo(catalog)
	ofertasRepo := store.NewOfertaRepo(catalog)
	for mercadoID, vitrine := range vitrines {
		if _, exists, err := catalog.GetMercado(ctx, mercadoID); err != nil {
			return err
		} else if !exists {
			continue
		}
		if err := coletarMercado(ctx, catalog, marcas, ofertasRepo, produto, mercadoID, vitrine, dia); err != nil {
			return err
		}
	}
	return nil
}

func coletarMercado(
	ctx context.Context,
	catalog *store.Catalog,
	marcas *store.MarcaRepo,
	ofertasRepo *store.OfertaRepo,
	produto store.Produto,
	mercadoID store.MercadoID,
	vitrine domain.Vitrine,
	dia string,
) error {
	coleta, err := coletaDoDia(ctx, catalog, produto.ID, mercadoID, dia)
	if err != nil {
		return err
	}
	cartoes, err := vitrine.Buscar(ctx, produto.Nome)
	if err != nil {
		coleta.Estado = store.EstadoFalhou
		coleta.UltimoErro = err.Error()
		return catalog.SaveColeta(ctx, coleta)
	}
	ofertas := make([]store.Oferta, 0, len(cartoes))
	descartes := 0
	for _, cartao := range cartoes {
		if !domain.CartaoCasaProduto(cartao.Nome, produto.NomeNorm) {
			continue
		}
		if !domain.CartaoCompleto(cartao) {
			descartes++
			continue
		}
		var marcaID *store.MarcaID
		if cartao.Marca != "" {
			marca, err := matchOrCreateMarca(ctx, marcas, cartao.Marca)
			if err != nil {
				return err
			}
			id := marca.ID
			marcaID = &id
		}
		ofertas = append(ofertas, domain.OfertaDaColeta(store.OfertaID(newID()), produto, mercadoID, marcaID, dia, cartao))
	}
	if err := ofertasRepo.SaveAllForColeta(ctx, coleta.ID, ofertas); err != nil {
		return err
	}
	coleta.Estado = store.EstadoAposValidacao(len(ofertas), descartes)
	coleta.UltimoErro = ""
	return catalog.SaveColeta(ctx, coleta)
}

func coletaDoDia(ctx context.Context, catalog *store.Catalog, produtoID store.ProdutoID, mercadoID store.MercadoID, dia string) (store.Coleta, error) {
	existing, ok, err := catalog.GetColetaByIdentity(ctx, produtoID, mercadoID, dia)
	if err != nil {
		return store.Coleta{}, err
	}
	if ok {
		existing.Estado = store.EstadoProcessando
		existing.UltimoErro = ""
		if err := catalog.SaveColeta(ctx, existing); err != nil {
			return store.Coleta{}, err
		}
		return existing, nil
	}
	c := store.Coleta{
		ID: store.ColetaID(newID()), ProdutoID: produtoID, MercadoID: mercadoID, Dia: dia,
		Estado: store.EstadoProcessando,
	}
	if err := catalog.SaveColeta(ctx, c); err != nil {
		return store.Coleta{}, err
	}
	return c, nil
}

func matchOrCreateMarca(ctx context.Context, repo *store.MarcaRepo, nome string) (store.Marca, error) {
	norm := domain.NormalizarRotulo(nome)
	existing, ok, err := repo.GetByNomeNorm(ctx, norm)
	if err != nil {
		return store.Marca{}, err
	}
	if ok {
		return existing, nil
	}
	m := store.Marca{ID: store.MarcaID(newID()), Nome: nome, NomeNorm: norm}
	if err := repo.Save(ctx, m); err != nil {
		return store.Marca{}, err
	}
	return m, nil
}
