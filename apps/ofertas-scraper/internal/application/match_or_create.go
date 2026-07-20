package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

// NewID mints an opaque entity id (ADR 0018).
func NewID() string {
	return uuid.NewString()
}

// PersistirOfertasValidas match-or-creates Produto/Marca and builds unique catalog Ofertas (ADR 0036).
func PersistirOfertasValidas(
	ctx context.Context,
	produtos domain.ProdutoRepository,
	marcas domain.MarcaRepository,
	ofertasRepo domain.OfertaRepository,
	doc domain.Documento,
	validas []domain.OfertaValidada,
) ([]domain.Oferta, error) {
	out := make([]domain.Oferta, 0, len(validas))
	for _, v := range validas {
		produto, err := matchOrCreateProduto(ctx, produtos, v)
		if err != nil {
			return nil, err
		}
		var marcaID *domain.MarcaID
		if v.Marca != "" {
			marca, err := matchOrCreateMarca(ctx, marcas, v.Marca)
			if err != nil {
				return nil, err
			}
			id := marca.ID
			marcaID = &id
		}
		candidate := domain.Oferta{
			ProdutoID:           produto.ID,
			MarcaID:             marcaID,
			MercadoID:           doc.MercadoID,
			Valor:               v.Valor,
			Quantidades:         v.Quantidades,
			Medida:              v.Medida,
			DataInicio:          v.DataInicio,
			DataExpiracao:       v.DataExpiracao,
			OrigemDataInicio:    v.OrigemDataInicio,
			OrigemDataExpiracao: v.OrigemDataExpiracao,
			Promocao:            v.Promocao,
			Comparativo:         v.Comparativo,
		}
		chave := domain.ChaveUnicaOferta(candidate)
		if ofertasRepo != nil {
			existing, ok, err := ofertasRepo.GetByUniq(ctx, chave)
			if err != nil {
				return nil, err
			}
			if ok {
				out = append(out, existing)
				continue
			}
		}
		candidate.ID = domain.OfertaID(NewID())
		out = append(out, candidate)
	}
	return out, nil
}

func matchOrCreateProduto(ctx context.Context, repo domain.ProdutoRepository, v domain.OfertaValidada) (domain.Produto, error) {
	norm := domain.NormalizarRotulo(v.Produto)
	existing, ok, err := repo.GetByNomeNorm(ctx, norm)
	if err != nil {
		return domain.Produto{}, err
	}
	if ok {
		merged := domain.UnirCategorias(existing.Categorias, v.Categorias)
		if len(merged) != len(existing.Categorias) {
			existing.Categorias = merged
			if err := repo.Save(ctx, existing); err != nil {
				return domain.Produto{}, err
			}
		}
		return existing, nil
	}
	p := domain.Produto{
		ID:         domain.ProdutoID(NewID()),
		Nome:       v.Produto,
		NomeNorm:   norm,
		Categorias: v.Categorias,
	}
	if err := repo.Save(ctx, p); err != nil {
		return domain.Produto{}, err
	}
	return p, nil
}

func matchOrCreateMarca(ctx context.Context, repo domain.MarcaRepository, nome string) (domain.Marca, error) {
	norm := domain.NormalizarRotulo(nome)
	existing, ok, err := repo.GetByNomeNorm(ctx, norm)
	if err != nil {
		return domain.Marca{}, err
	}
	if ok {
		return existing, nil
	}
	m := domain.Marca{
		ID:       domain.MarcaID(NewID()),
		Nome:     nome,
		NomeNorm: norm,
	}
	if err := repo.Save(ctx, m); err != nil {
		return domain.Marca{}, err
	}
	return m, nil
}
