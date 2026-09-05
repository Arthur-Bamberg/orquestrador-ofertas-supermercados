package shopfully

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
)

type publicationDescriptor struct {
	BundlePath string `json:"bundlePath"`
}

type publicationRoot struct {
	PublicationDescriptor publicationDescriptor `json:"publicationDescriptor"`
}

type pagesRoot struct {
	PublicationDescriptor pagesDescriptor `json:"publicationDescriptor"`
}

type pagesDescriptor struct {
	NumberOfPages   int              `json:"numberOfPages"`
	PageDescriptors []pageDescriptor `json:"pageDescriptors"`
}

type pageDescriptor struct {
	BundlePath string `json:"bundlePath"`
	BundlePart int    `json:"bundlePart"`
}

type pageBody struct {
	PageNumber                    int                           `json:"pageNumber"`
	PageRepresentationDescriptors []pageRepresentationDescriptor `json:"pageRepresentationDescriptors"`
}

type pageRepresentationDescriptor struct {
	PageRepresentation struct {
		ResourcePath string `json:"resourcePath"`
	} `json:"pageRepresentation"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
	Type   string `json:"type"`
}

func absoluteURL(path string) string {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") {
		return path
	}
	path = strings.TrimPrefix(path, "//")
	return "https://" + path
}

// PublicationBundlePath reads the first pages bundle URL from a publication descriptor.
func PublicationBundlePath(raw []byte) (string, error) {
	var root publicationRoot
	if err := json.Unmarshal(raw, &root); err != nil {
		return "", err
	}
	if root.PublicationDescriptor.BundlePath == "" {
		return "", fmt.Errorf("bundlePath vazio")
	}
	return absoluteURL(root.PublicationDescriptor.BundlePath), nil
}

// BundlePaths returns unique page-bundle URLs in listing order.
func BundlePaths(raw []byte) ([]string, error) {
	var root pagesRoot
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	var out []string
	for _, d := range root.PublicationDescriptor.PageDescriptors {
		if d.BundlePath == "" {
			continue
		}
		u := absoluteURL(d.BundlePath)
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("nenhum bundlePath")
	}
	return out, nil
}

// BestPageImages picks the highest-resolution image URL per page in a bundle JSON.
func BestPageImages(raw []byte) ([]domain.Pagina, error) {
	var blob map[string]json.RawMessage
	if err := json.Unmarshal(raw, &blob); err != nil {
		return nil, err
	}
	var out []domain.Pagina
	for key, val := range blob {
		if key == "publicationDescriptor" {
			continue
		}
		if _, err := strconv.Atoi(key); err != nil {
			continue
		}
		var page pageBody
		if err := json.Unmarshal(val, &page); err != nil {
			return nil, fmt.Errorf("pagina %s: %w", key, err)
		}
		url, ok := bestImageURL(page)
		if !ok {
			continue
		}
		n := page.PageNumber
		if n == 0 {
			n, _ = strconv.Atoi(key)
		}
		out = append(out, domain.Pagina{Numero: n, URL: url})
	}
	out = sortPaginas(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("nenhuma imagem de página")
	}
	return out, nil
}

func sortPaginas(pages []domain.Pagina) []domain.Pagina {
	sort.Slice(pages, func(i, j int) bool { return pages[i].Numero < pages[j].Numero })
	return pages
}

func bestImageURL(page pageBody) (string, bool) {
	bestArea := -1
	bestURL := ""
	for _, d := range page.PageRepresentationDescriptors {
		if d.Type != "image" {
			continue
		}
		path := d.PageRepresentation.ResourcePath
		if path == "" {
			continue
		}
		area := d.Width * d.Height
		if area > bestArea {
			bestArea = area
			bestURL = absoluteURL(path)
		}
	}
	if bestURL == "" {
		return "", false
	}
	return bestURL, true
}
