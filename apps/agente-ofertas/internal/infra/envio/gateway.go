package envio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Gateway struct {
	Base   string
	Token  string
	Client *http.Client
}

func (g Gateway) Enviar(ctx context.Context, conversaJID, corpo string) error {
	if g.Client == nil {
		g.Client = &http.Client{Timeout: 30 * time.Second}
	}
	body, err := json.Marshal(map[string]string{"conversaJid": conversaJID, "corpo": corpo})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(g.Base, "/")+"/envios", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.Token)
	res, err := g.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("envios: HTTP %d", res.StatusCode)
	}
	return nil
}
