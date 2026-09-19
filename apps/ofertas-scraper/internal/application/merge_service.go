package application

import (
	"context"
	"fmt"
	"log"
	"sort"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

type productCatalog interface {
	ListProdutos(ctx context.Context) ([]store.Produto, error)
	SaveProduto(ctx context.Context, p store.Produto) error
	DeleteProduto(ctx context.Context, id store.ProdutoID) error
}

type offerRepository interface {
	ListOfertas(ctx context.Context) ([]store.Oferta, error)
	SaveOferta(ctx context.Context, o store.Oferta, documentoIDs []store.DocumentoID) error
	SaveOfertasForColeta(ctx context.Context, coletaID store.ColetaID, ofertas []store.Oferta) error
	UpdateOfertaProdutoID(ctx context.Context, ofertaID store.OfertaID, newProdutoID store.ProdutoID) error
	GetOfertaSources(ctx context.Context, ofertaID store.OfertaID) ([]store.DocumentoID, []store.ColetaID, error)
}

type MergeService struct {
	cat        productCatalog
	ofertaRepo offerRepository
	// TODO: Add Gemini client for sanitization if needed
}

func NewMergeService(cat *store.Catalog) *MergeService {
	return &MergeService{cat: cat, ofertaRepo: cat}
}

func (s *MergeService) MergeProducts(ctx context.Context) error {
	log.Println("Starting product merge...")

	products, err := s.cat.ListProdutos(ctx)
	if err != nil {
		return fmt.Errorf("failed to list products: %w", err)
	}

	productMap := make(map[store.ProdutoID]store.Produto)
	for _, p := range products {
		productMap[p.ID] = p
	}

	// 1. Sanitize product names and set OriginalNome if empty
	for i := range products {
		p := &products[i]

		if p.OriginalNome == "" {
			p.OriginalNome = p.Nome
		}

		cleanedName := store.NormalizarRotulo(p.OriginalNome)
		nomeNorm := store.NormalizarRotulo(cleanedName)

		if p.Nome != cleanedName || p.NomeNorm != nomeNorm {
			log.Printf("Sanitizing product %s: \"%s\" -> \"%s\" (norm: \"%s\" -> \"%s\")", p.ID, p.Nome, cleanedName, p.NomeNorm, nomeNorm)
			p.Nome = cleanedName
			p.NomeNorm = nomeNorm
			if err := s.cat.SaveProduto(ctx, *p); err != nil {
				return fmt.Errorf("failed to save product %s: %w", p.ID, err)
			}
		}
	}

	// 2. Backfill size for offers if the product name had grammage and offer was 1 unit
	offers, err := s.ofertaRepo.ListOfertas(ctx)
	if err != nil {
		return fmt.Errorf("failed to list offers: %w", err)
	}

	for _, o := range offers {
		if o.Medida == store.MedidaUnidade && len(o.Quantidades) == 1 && o.Quantidades[0] == 1 {
			p, ok := productMap[o.ProdutoID]
			if !ok {
				log.Printf("Product %s not found for offer %s, skipping backfill", o.ProdutoID, o.ID)
				continue
			}

			if q, m, ok := store.ExtractQuantityAndMeasure(p.OriginalNome); ok {
				log.Printf("Backfilling offer %s: \"%v %s\" -> \"%v %s\" (from original product name \"%s\")", o.ID, o.Quantidades, o.Medida, []float64{q}, m, p.OriginalNome)

				o.Quantidades = []float64{q}
				o.Medida = m

				docIDs, coletaIDs, err := s.ofertaRepo.GetOfertaSources(ctx, o.ID)
				if err != nil {
					log.Printf("Failed to get sources for offer %s: %v", o.ID, err)
					continue
				}

				if len(docIDs) > 0 {
					// Re-save using the original document IDs
					if err := s.ofertaRepo.SaveOferta(ctx, o, docIDs); err != nil {
						log.Printf("Failed to save backfilled offer %s (documento): %v", o.ID, err)
					}
				} else if len(coletaIDs) > 0 {
					// Re-save using the original coleta ID (assuming one coleta per offer for simplicity)
					if err := s.ofertaRepo.SaveOfertasForColeta(ctx, coletaIDs[0], []store.Oferta{o}); err != nil {
						log.Printf("Failed to save backfilled offer %s (coleta): %v", o.ID, err)
					}
				} else {
					log.Printf("Offer %s has no associated document or coleta sources. Skipping backfill save.", o.ID)
				}
			} else {
				log.Printf("Original product name \"%s\" did not contain quantity/measure for offer %s. Skipping backfill.", p.OriginalNome, o.ID)
			}
		}
	}

	// 3. Group products by NomeNorm for merging
	productsByNomeNorm := make(map[string][]store.Produto)
	for _, p := range products {
		productsByNomeNorm[p.NomeNorm] = append(productsByNomeNorm[p.NomeNorm], p)
	}

	for nomeNorm, prods := range productsByNomeNorm {
		if len(prods) <= 1 {
			continue // No merging needed if only one product with this NomeNorm
		}

		// Sort products to ensure deterministic canonical selection (e.g., by ID)
		sort.Slice(prods, func(i, j int) bool {
			return prods[i].ID < prods[j].ID
		})

		canonicalProduct := prods[0]
		log.Printf("Merging products for NomeNorm \"%s\": canonical is %s (\"%s\")", nomeNorm, canonicalProduct.ID, canonicalProduct.Nome)

		for _, duplicateProduct := range prods[1:] {
			log.Printf("  - Transferring offers from duplicate product %s (\"%s\") to canonical %s", duplicateProduct.ID, duplicateProduct.Nome, canonicalProduct.ID)
			
			// Find and update offers associated with the duplicate product
			allOffers, err := s.ofertaRepo.ListOfertas(ctx)
			if err != nil {
				return fmt.Errorf("failed to list offers for merging: %w", err)
			}
			for _, offer := range allOffers {
				if offer.ProdutoID == duplicateProduct.ID {
					log.Printf("    Updating offer %s from %s to %s", offer.ID, offer.ProdutoID, canonicalProduct.ID)
					if err := s.ofertaRepo.UpdateOfertaProdutoID(ctx, offer.ID, canonicalProduct.ID); err != nil {
						log.Printf("Failed to update ProdutoID for offer %s: %v", offer.ID, err)
						// Depending on requirements, might want to return an error or continue
					}
				}
			}

			log.Printf("  - Deleting duplicate product %s (\"%s\")", duplicateProduct.ID, duplicateProduct.Nome)
			if err := s.cat.DeleteProduto(ctx, duplicateProduct.ID); err != nil {
				log.Printf("Failed to delete duplicate product %s: %v", duplicateProduct.ID, err)
				// Depending on requirements, might want to return an error or continue
			}
		}
	}

	fmt.Println("Product merge completed: products consolidated by normalized name.")

	return nil
}
