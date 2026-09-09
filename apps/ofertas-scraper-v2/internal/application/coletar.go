package application

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func newID() string { return uuid.NewString() }

var coletaInflight sync.Map // key termo|mercado|dia → *inflightColeta

type inflightColeta struct {
	done chan struct{}
	mu   sync.Mutex
	res  []store.Oferta
	err  error
}

func coletaKey(termo string, mercadoID store.MercadoID, dia string) string {
	return termo + "|" + string(mercadoID) + "|" + dia
}

// Coletar searches each Vitrine with the Item term, persists every complete card as Oferta, and returns what it wrote.
func Coletar(ctx context.Context, catalog *store.Catalog, vitrines map[store.MercadoID]domain.Vitrine, termo, dia string) ([]store.Oferta, error) {
	termo = store.NormalizarRotulo(termo)
	if termo == "" {
		return nil, fmt.Errorf("%w: termo da Coleta obrigatório", store.ErrInvalid)
	}
	produtos := store.NewProdutoRepo(catalog)
	marcas := store.NewMarcaRepo(catalog)
	ofertasRepo := store.NewOfertaRepo(catalog)
	var out []store.Oferta
	for mercadoID, vitrine := range vitrines {
		if _, exists, err := catalog.GetMercado(ctx, mercadoID); err != nil {
			return nil, err
		} else if !exists {
			continue
		}
		ofertas, err := coletarMercado(ctx, catalog, produtos, marcas, ofertasRepo, mercadoID, vitrine, termo, dia)
		if err != nil {
			return nil, err
		}
		out = append(out, ofertas...)
	}
	return out, nil
}

func coletarMercado(
	ctx context.Context,
	catalog *store.Catalog,
	produtos *store.ProdutoRepo,
	marcas *store.MarcaRepo,
	ofertasRepo *store.OfertaRepo,
	mercadoID store.MercadoID,
	vitrine domain.Vitrine,
	termo, dia string,
) ([]store.Oferta, error) {
	key := coletaKey(termo, mercadoID, dia)
	if existing, ok, err := catalog.GetColetaByIdentity(ctx, termo, mercadoID, dia); err != nil {
		return nil, err
	} else if ok && !store.DeveReprocessar(existing.Estado) {
		return ofertasRepo.ListByColeta(ctx, existing.ID)
	}

	slot := &inflightColeta{done: make(chan struct{})}
	if v, loaded := coletaInflight.LoadOrStore(key, slot); loaded {
		wait := v.(*inflightColeta)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-wait.done:
			wait.mu.Lock()
			defer wait.mu.Unlock()
			return wait.res, wait.err
		}
	}
	defer func() {
		close(slot.done)
		coletaInflight.Delete(key)
	}()

	ofertas, err := executarColeta(ctx, catalog, produtos, marcas, ofertasRepo, mercadoID, vitrine, termo, dia)
	slot.mu.Lock()
	slot.res, slot.err = ofertas, err
	slot.mu.Unlock()
	return ofertas, err
}

func executarColeta(
	ctx context.Context,
	catalog *store.Catalog,
	produtos *store.ProdutoRepo,
	marcas *store.MarcaRepo,
	ofertasRepo *store.OfertaRepo,
	mercadoID store.MercadoID,
	vitrine domain.Vitrine,
	termo, dia string,
) ([]store.Oferta, error) {
	coleta, err := coletaDoDia(ctx, catalog, termo, mercadoID, dia)
	if err != nil {
		return nil, err
	}
	resultado, err := vitrine.Buscar(ctx, termo)
	if err != nil {
		coleta.Estado = store.EstadoFalhou
		coleta.UltimoErro = err.Error()
		if saveErr := catalog.SaveColeta(ctx, coleta); saveErr != nil {
			return nil, saveErr
		}
		return nil, nil
	}
	ofertas := make([]store.Oferta, 0, len(resultado.Cartoes))
	descartes := 0
	for _, cartao := range resultado.Cartoes {
		if !domain.CartaoCompleto(cartao) {
			descartes++
			continue
		}
		produtoNome := domain.ProdutoDoCartao(cartao.Nome, cartao.Marca)
		if produtoNome == "" {
			descartes++
			continue
		}
		produto, err := store.MatchOrCreateProduto(ctx, produtos, produtoNome, store.ProdutoID(newID()))
		if err != nil {
			return nil, err
		}
		var marcaID *store.MarcaID
		if cartao.Marca != "" {
			marca, err := store.MatchOrCreateMarca(ctx, marcas, cartao.Marca, store.MarcaID(newID()))
			if err != nil {
				return nil, err
			}
			id := marca.ID
			marcaID = &id
		}
		ofertas = append(ofertas, domain.OfertaDaColeta(store.OfertaID(newID()), produto.ID, mercadoID, marcaID, dia, cartao))
	}
	if err := ofertasRepo.SaveAllForColeta(ctx, coleta.ID, ofertas); err != nil {
		return nil, err
	}
	coleta.Estado = domain.EstadoAposColeta(len(ofertas), descartes, resultado.Esgotada)
	coleta.UltimoErro = ""
	if err := catalog.SaveColeta(ctx, coleta); err != nil {
		return nil, err
	}
	return ofertas, nil
}

func coletaDoDia(ctx context.Context, catalog *store.Catalog, termo string, mercadoID store.MercadoID, dia string) (store.Coleta, error) {
	existing, ok, err := catalog.GetColetaByIdentity(ctx, termo, mercadoID, dia)
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
		ID: store.ColetaID(newID()), Termo: termo, MercadoID: mercadoID, Dia: dia,
		Estado: store.EstadoProcessando,
	}
	if err := catalog.SaveColeta(ctx, c); err != nil {
		return store.Coleta{}, err
	}
	return c, nil
}
