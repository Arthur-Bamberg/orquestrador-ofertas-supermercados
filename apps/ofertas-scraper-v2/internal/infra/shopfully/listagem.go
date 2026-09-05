package shopfully

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
)

var flyerCardRe = regexp.MustCompile(`(?s)<div id="(\d+)" class="flyerCard[^"]*"[^>]*>.*?</div>\s*</div>\s*</div>`)
var flyerHrefRe = regexp.MustCompile(`href='([^']*flyerId=\d+[^']*)'`)
var flyerTitleRe = regexp.MustCompile(`<h3 class="flyerCard__titleText">([^<]+)</h3>`)
var flyerVenceRe = regexp.MustCompile(`<span class="flyerCard__detailsPrimary">([^<]+)</span>`)
var flyerTypeRe = regexp.MustCompile(`data-type='flyer'`)

// ParseListagem extracts unique Encartes from a Shopfully category listing HTML.
func ParseListagem(htmlBody string) ([]domain.Encarte, error) {
	seen := map[string]struct{}{}
	var out []domain.Encarte
	for _, m := range flyerCardRe.FindAllStringSubmatch(htmlBody, -1) {
		block := m[0]
		if !flyerTypeRe.MatchString(block) {
			continue
		}
		id := m[1]
		if _, ok := seen[id]; ok {
			continue
		}
		href := flyerHrefRe.FindStringSubmatch(block)
		if href == nil {
			return nil, fmt.Errorf("encarte %s sem viewer path", id)
		}
		title := flyerTitleRe.FindStringSubmatch(block)
		if title == nil {
			return nil, fmt.Errorf("encarte %s sem mercado", id)
		}
		vence := ""
		if v := flyerVenceRe.FindStringSubmatch(block); v != nil {
			vence = strings.TrimSpace(html.UnescapeString(v[1]))
		}
		path := html.UnescapeString(href[1])
		seen[id] = struct{}{}
		out = append(out, domain.Encarte{
			ID:         id,
			Mercado:    strings.TrimSpace(html.UnescapeString(title[1])),
			ViewerPath: path,
			Vencimento: vence,
		})
	}
	return out, nil
}
