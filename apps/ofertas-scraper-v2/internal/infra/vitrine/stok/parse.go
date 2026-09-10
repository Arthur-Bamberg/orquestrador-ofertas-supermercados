package stok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/domain"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/infra/vitrine/osuper"
)

const (
	APIBase              = "https://services.vipcommerce.com.br/api-admin/v1"
	OrgID                = "130"
	FilialID             = "1" // site default (Avenida Zeferino Costa, 140)
	CentroDistribuicaoID = "1" // centro_distribuicao_padrao_id
	DomainKey            = "stokonline.com.br"
	PageSize             = 100
	OrganizationIDHeader = "OrganizationId"
	DomainKeyHeader      = "DomainKey"
	lojaUser             = "loja"
	lojaAuthJWT          = "df072f85df9bf7dd71b6811c34bdbaa4f219d98775b56cff9dfa5f8ca1bf8469" // public VipCommerce frontend env
)

func SearchURL(query string, from, size int) string {
	if size <= 0 {
		size = PageSize
	}
	page := from/size + 1
	term := encodeTermo(query)
	return APIBase + "/org/" + OrgID + "/filial/" + FilialID + "/centro_distribuicao/" + CentroDistribuicaoID +
		"/loja/buscas/produtos/termo/" + term + "?page=" + strconv.Itoa(page)
}

func encodeTermo(query string) string {
	q := strings.TrimSpace(query)
	q = strings.NewReplacer(`\`, "+", "/", "+", " ", "+", "%", "+", "?", "+", "=", "+").Replace(q)
	return url.PathEscape(q)
}

type searchDoc struct {
	Data struct {
		Produtos []hit `json:"produtos"`
	} `json:"data"`
	Paginator struct {
		Page         int `json:"page"`
		ItemsPerPage int `json:"items_per_page"`
		TotalPages   int `json:"total_pages"`
		TotalItems   int `json:"total_items"`
	} `json:"paginator"`
}

type hit struct {
	Descricao                  string    `json:"descricao"`
	Marca                      marcaJSON `json:"marca"`
	Preco                      flexFloat `json:"preco"`
	PrecoOriginal              flexFloat `json:"preco_original"`
	EmOferta                   bool      `json:"em_oferta"`
	Oferta                     *oferta   `json:"oferta"`
	UnidadeSigla               string    `json:"unidade_sigla"`
	QuantidadeUnidadeDiferente flexFloat `json:"quantidade_unidade_diferente"`
}

type oferta struct {
	PrecoOferta flexFloat `json:"preco_oferta"`
}

type marcaJSON struct {
	Nome string
}

func (m *marcaJSON) UnmarshalJSON(b []byte) error {
	if string(b) == "null" || len(b) == 0 {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		m.Nome = s
		return nil
	}
	var obj struct {
		Nome string `json:"nome"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return err
	}
	m.Nome = obj.Nome
	return nil
}

type flexFloat float64

func (f *flexFloat) UnmarshalJSON(b []byte) error {
	if string(b) == "null" || len(b) == 0 {
		*f = 0
		return nil
	}
	var n float64
	if err := json.Unmarshal(b, &n); err == nil {
		*f = flexFloat(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		*f = 0
		return nil
	}
	n, err := strconv.ParseFloat(s, 64)
	*f = flexFloat(n)
	return err
}

func ParseSearch(body []byte) (domain.PaginaVitrine, error) {
	var doc searchDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return domain.PaginaVitrine{}, err
	}
	out := make([]domain.CartaoVitrine, 0, len(doc.Data.Produtos))
	for _, h := range doc.Data.Produtos {
		frac := float64(h.QuantidadeUnidadeDiferente)
		if frac <= 0 {
			frac = 1
		}
		medida, qty := osuper.MedidaDoCartao(h.UnidadeSigla, frac)
		valor := float64(h.Preco)
		ofertaValor := 0.0
		if h.Oferta != nil {
			ofertaValor = float64(h.Oferta.PrecoOferta)
		}
		if ofertaValor > 0 {
			valor = ofertaValor
		}
		unit := strings.ToUpper(strings.TrimSpace(h.UnidadeSigla))
		if (unit == "KG" || unit == "L" || unit == "LT") && qty == 1000 {
			if !h.EmOferta && float64(h.PrecoOriginal) > 0 {
				valor = float64(h.PrecoOriginal)
			} else if frac != 1 {
				valor = valor / frac
			}
		}
		promo := h.EmOferta || (ofertaValor > 0 && ofertaValor < float64(h.Preco))
		out = append(out, domain.CartaoVitrine{
			Nome:                 h.Descricao,
			Marca:                h.Marca.Nome,
			Valor:                valor,
			Quantidades:          []float64{qty},
			Medida:               medida,
			IndicacaoPromocional: promo,
		})
	}
	pageSize := doc.Paginator.ItemsPerPage
	if pageSize <= 0 {
		pageSize = PageSize
	}
	return domain.PaginaVitrine{
		Cartoes:  out,
		Total:    doc.Paginator.TotalItems,
		NextFrom: doc.Paginator.Page * pageSize,
		HasNext:  doc.Paginator.Page < doc.Paginator.TotalPages,
	}, nil
}

func loginURL() string {
	return APIBase + "/org/" + OrgID + "/auth/loja/login"
}

func defaultAPIHeaders() http.Header {
	h := make(http.Header)
	h.Set(OrganizationIDHeader, OrgID)
	h.Set(DomainKeyHeader, DomainKey)
	h.Set("Accept", "application/json")
	h.Set("Content-Type", "application/json")
	return h
}

// TokenProvider fetches and caches the VipCommerce loja JWT.
type TokenProvider struct {
	HTTP     *http.Client
	LoginURL string

	mu    sync.Mutex
	token string
	until time.Time
}

func (p *TokenProvider) Header(ctx context.Context) (http.Header, error) {
	tok, err := p.Token(ctx)
	if err != nil {
		return nil, err
	}
	h := defaultAPIHeaders()
	h.Set("Authorization", "Bearer "+tok)
	return h, nil
}

func (p *TokenProvider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.token != "" && time.Now().Before(p.until) {
		return p.token, nil
	}
	httpClient := p.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	endpoint := p.LoginURL
	if endpoint == "" {
		endpoint = loginURL()
	}
	payload, err := json.Marshal(map[string]string{
		"domain":   DomainKey,
		"username": lojaUser,
		"key":      lojaAuthJWT,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header = defaultAPIHeaders()
	req.Header.Set("User-Agent", "ofertas-scraper-v2/1.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("stok login HTTP %d", resp.StatusCode)
	}
	var doc struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", err
	}
	if doc.Data == "" {
		return "", fmt.Errorf("stok loja JWT not found")
	}
	p.token = doc.Data
	p.until = time.Now().Add(30 * time.Minute)
	return p.token, nil
}
