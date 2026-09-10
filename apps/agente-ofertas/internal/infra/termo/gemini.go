package termo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	geminiDefaultBase  = "https://generativelanguage.googleapis.com"
	geminiDefaultModel = "gemini-3-flash-preview"
	instrucaoTermo     = `Você extrai o Termo de busca de uma linha de lista de supermercado.
Devolva só JSON: {"termo":"..."}.
O termo tem o tipo vendável, cultivar ou processo se a pessoa disse (orgânica, italiana) e a marca.
Não inclua verbo de compra, cumprimento, quantidade, unidade de medida, artigo, preposição nem pontuação.
Não invente o que a pessoa não disse. Não use catálogo.
Se não restar nada de mercearia, {"termo":""}.`
)

type Gemini struct {
	Key   string
	Model string
	Base  string
	HTTP  *http.Client
}

func (g Gemini) Termo(ctx context.Context, item string) (string, error) {
	client := g.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	payload, err := json.Marshal(map[string]any{
		"systemInstruction": map[string]any{
			"parts": []map[string]string{{"text": instrucaoTermo}},
		},
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": item}}},
		},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
			"temperature":      0,
		},
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint(), bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("gemini termo: HTTP %d", res.StatusCode)
	}
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini termo: resposta vazia")
	}
	raw := strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text)
	var out struct {
		Termo string `json:"termo"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return "", err
	}
	return strings.Join(strings.Fields(strings.TrimSpace(out.Termo)), " "), nil
}

func (g Gemini) endpoint() string {
	base := strings.TrimRight(g.Base, "/")
	if base == "" {
		base = geminiDefaultBase
	}
	model := g.Model
	if model == "" {
		model = geminiDefaultModel
	}
	return base + "/v1beta/models/" + url.PathEscape(model) + ":generateContent?key=" + url.QueryEscape(g.Key)
}
