package upstash

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client talks to Upstash Redis REST (or SRH locally) — ADR 0001 / 0019.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		httpClient: httpClient,
	}
}

func (c *Client) Do(ctx context.Context, args ...any) (json.RawMessage, error) {
	body, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("upstash http %d: %s", res.StatusCode, string(raw))
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("upstash decode: %w (body=%s)", err, string(raw))
	}
	if envelope.Error != "" {
		return nil, fmt.Errorf("upstash: %s", envelope.Error)
	}
	return envelope.Result, nil
}

func (c *Client) Get(ctx context.Context, key string) (string, bool, error) {
	result, err := c.Do(ctx, "GET", key)
	if err != nil {
		return "", false, err
	}
	if string(result) == "null" {
		return "", false, nil
	}
	var s string
	if err := json.Unmarshal(result, &s); err != nil {
		return "", false, err
	}
	return s, true, nil
}

func (c *Client) Set(ctx context.Context, key, value string) error {
	_, err := c.Do(ctx, "SET", key, value)
	return err
}

// SetEX sets key with TTL in seconds (Upstash SET … EX).
func (c *Client) SetEX(ctx context.Context, key, value string, ttlSeconds int) error {
	_, err := c.Do(ctx, "SET", key, value, "EX", ttlSeconds)
	return err
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	args := make([]any, 0, 1+len(keys))
	args = append(args, "DEL")
	for _, k := range keys {
		args = append(args, k)
	}
	_, err := c.Do(ctx, args...)
	return err
}

func (c *Client) SAdd(ctx context.Context, key, member string) error {
	_, err := c.Do(ctx, "SADD", key, member)
	return err
}

func (c *Client) SRem(ctx context.Context, key, member string) error {
	_, err := c.Do(ctx, "SREM", key, member)
	return err
}

func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	result, err := c.Do(ctx, "SMEMBERS", key)
	if err != nil {
		return nil, err
	}
	if string(result) == "null" {
		return nil, nil
	}
	var members []string
	if err := json.Unmarshal(result, &members); err != nil {
		return nil, err
	}
	return members, nil
}
