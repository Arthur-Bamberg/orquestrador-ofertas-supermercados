package coleta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	store "github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

type Cliente struct {
	Base   string
	Token  string
	Client *http.Client
}

func (c Cliente) Coletar(ctx context.Context, termo string) ([]store.Oferta, error) {
	httpClient := c.Client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Minute}
	}
	payload, err := json.Marshal(map[string]string{"termo": termo})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.Base, "/")+"/coletas", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = res.Status
		}
		return nil, fmt.Errorf("coleta HTTP %d: %s", res.StatusCode, msg)
	}
	var out struct {
		Ofertas []store.Oferta `json:"ofertas"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out.Ofertas == nil {
		out.Ofertas = []store.Oferta{}
	}
	return out.Ofertas, nil
}
