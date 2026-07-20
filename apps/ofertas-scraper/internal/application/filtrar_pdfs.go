package application

import (
	"regexp"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

// FiltrarPDFs applies Fonte.FiltroNomeDocumento as regex on filename (ADR 0020).
// Empty filter keeps all. Invalid regex returns error.
func FiltrarPDFs(fonte domain.Fonte, all []domain.PDFDescoberto) (kept, rejected []domain.PDFDescoberto, err error) {
	if fonte.FiltroNomeDocumento == "" {
		return all, nil, nil
	}
	re, err := regexp.Compile(fonte.FiltroNomeDocumento)
	if err != nil {
		return nil, nil, err
	}
	for _, p := range all {
		if re.MatchString(p.Filename) {
			kept = append(kept, p)
		} else {
			rejected = append(rejected, p)
		}
	}
	return kept, rejected, nil
}
