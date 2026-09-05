package agente

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Cliente struct {
	URL    string
	Token  string
	Client *http.Client
}

func (c *Cliente) Atender(ctx context.Context, conversaJID, corpo string) error {
	httpClient := c.Client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 45 * time.Second}
	}
	body, err := json.Marshal(map[string]string{"conversaJid": conversaJID, "corpo": corpo})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.URL, "/")+"/listas", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("agente: HTTP %d", res.StatusCode)
	}
	return nil
}
