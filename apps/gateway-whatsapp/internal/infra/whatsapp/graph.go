package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/domain"
)

const graphDefaultBase = "https://graph.facebook.com"
const graphDefaultVersion = "v21.0"

type Cliente struct {
	HTTP          *http.Client
	BaseURL       string
	Version       string
	Token         string
	PhoneNumberID string
	DisplayJID    domain.JID
}

func (c *Cliente) Pronto() bool {
	return c != nil && strings.TrimSpace(c.Token) != "" && strings.TrimSpace(c.PhoneNumberID) != ""
}

func (c *Cliente) Situacao() domain.CanalSituacao {
	if !c.Pronto() {
		return domain.CanalSituacao{Estado: domain.CanalNaoConfigurado}
	}
	return domain.CanalSituacao{Estado: domain.CanalPronto, JID: c.DisplayJID}
}

func (c *Cliente) Enviar(ctx context.Context, e domain.Envio) (string, error) {
	if !c.Pronto() {
		return "", domain.ErrCanalNaoConfigurado
	}
	to := Telefone(e.Destino)
	if to == "" {
		return "", fmt.Errorf("destino inválido")
	}
	body := map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
	}
	switch {
	case e.Template != nil && strings.TrimSpace(e.Template.Nome) != "":
		idioma := strings.TrimSpace(e.Template.Idioma)
		if idioma == "" {
			idioma = "pt_BR"
		}
		body["type"] = "template"
		body["template"] = map[string]any{
			"name":     e.Template.Nome,
			"language": map[string]string{"code": idioma},
		}
	case e.Midia != nil && len(e.Midia.Conteudo) > 0:
		mediaID, err := c.uploadMedia(ctx, e.Midia)
		if err != nil {
			return "", err
		}
		tipo, field := graphMediaField(e.Midia.Tipo)
		payload := map[string]any{"id": mediaID}
		if strings.TrimSpace(e.Corpo) != "" && (e.Midia.Tipo == domain.MidiaImagem || e.Midia.Tipo == domain.MidiaVideo || e.Midia.Tipo == domain.MidiaDocumento) {
			payload["caption"] = e.Corpo
		}
		if e.Midia.Tipo == domain.MidiaDocumento && e.Midia.Filename != "" {
			payload["filename"] = e.Midia.Filename
		}
		body["type"] = tipo
		body[field] = payload
	default:
		body["type"] = "text"
		body["text"] = map[string]any{"body": e.Corpo, "preview_url": false}
	}
	id, err := c.postMessages(ctx, body)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (c *Cliente) BaixarMidia(ctx context.Context, id string, tipo domain.TipoMidia, mime, filename string) (*domain.MidiaBytes, error) {
	if !c.Pronto() || id == "" {
		return &domain.MidiaBytes{Tipo: tipo, MIME: mime, Filename: filename}, nil
	}
	meta, err := c.getJSON(ctx, c.url("/"+id))
	if err != nil {
		return &domain.MidiaBytes{Tipo: tipo, MIME: mime, Filename: filename}, err
	}
	rawURL, _ := meta["url"].(string)
	if mime == "" {
		mime, _ = meta["mime_type"].(string)
	}
	stub := &domain.MidiaBytes{Tipo: tipo, MIME: mime, Filename: filename}
	if rawURL == "" {
		return stub, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return stub, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	res, err := c.http().Do(req)
	if err != nil {
		return stub, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return stub, err
	}
	if res.StatusCode >= 300 {
		return stub, fmt.Errorf("graph media %s: %s", res.Status, truncate(b))
	}
	stub.Conteudo = b
	return stub, nil
}

func (c *Cliente) postMessages(ctx context.Context, body map[string]any) (string, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/"+c.PhoneNumberID+"/messages"), bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http().Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode >= 300 {
		return "", graphError(b)
	}
	var out struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return "", err
	}
	if len(out.Messages) == 0 || out.Messages[0].ID == "" {
		return "", fmt.Errorf("graph: resposta sem id")
	}
	return out.Messages[0].ID, nil
}

func (c *Cliente) uploadMedia(ctx context.Context, midia *domain.MidiaBytes) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("messaging_product", "whatsapp")
	mime := midia.MIME
	if mime == "" {
		mime = "application/octet-stream"
	}
	_ = w.WriteField("type", mime)
	name := midia.Filename
	if name == "" {
		name = "arquivo"
	}
	part, err := w.CreateFormFile("file", name)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(midia.Conteudo); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/"+c.PhoneNumberID+"/media"), &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", w.FormDataContentType())
	res, err := c.http().Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode >= 300 {
		return "", graphError(b)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", fmt.Errorf("graph: upload sem id")
	}
	return out.ID, nil
}

func (c *Cliente) getJSON(ctx context.Context, rawURL string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	res, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, graphError(b)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Cliente) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Cliente) url(path string) string {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = graphDefaultBase
	}
	ver := strings.TrimSpace(c.Version)
	if ver == "" {
		ver = graphDefaultVersion
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + "/" + ver + path
}

func graphMediaField(t domain.TipoMidia) (tipo, field string) {
	switch t {
	case domain.MidiaImagem, domain.MidiaFigurinha:
		return "image", "image"
	case domain.MidiaAudio:
		return "audio", "audio"
	case domain.MidiaVideo:
		return "video", "video"
	case domain.MidiaDocumento:
		return "document", "document"
	default:
		return "document", "document"
	}
}

func graphError(body []byte) error {
	var wrap struct {
		Error struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &wrap); err == nil && wrap.Error.Code != 0 {
		if wrap.Error.Code == 131047 {
			return domain.ErrForaDaJanela
		}
		msg := wrap.Error.Message
		if msg == "" {
			msg = fmt.Sprintf("graph code %d", wrap.Error.Code)
		}
		return errors.New(msg)
	}
	return fmt.Errorf("graph: %s", truncate(body))
}

func Telefone(jid domain.JID) string {
	n := domain.NormalizarJID(string(jid))
	user, _, ok := strings.Cut(string(n), "@")
	if !ok {
		return string(n)
	}
	return user
}

func truncate(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 300 {
		return s[:300]
	}
	return s
}
