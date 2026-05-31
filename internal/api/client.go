package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
	"github.com/shhac/agent-cloudflare/internal/output"
)

const defaultBaseURL = "https://api.cloudflare.com/client/v4"

type Client struct {
	baseURL string
	token   string
	http    *http.Client
	debug   bool
}

type Options struct {
	Token   string
	BaseURL string
}

func NewClient(opts Options) *Client {
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   opts.Token,
		http:    &http.Client{},
	}
}

func (c *Client) SetDebug(enabled bool) {
	c.debug = enabled
}

func (c *Client) Get(ctx context.Context, path string, params url.Values) (json.RawMessage, *ResultInfo, error) {
	return c.do(ctx, http.MethodGet, buildPath(path, params), nil)
}

func (c *Client) RawRequest(ctx context.Context, method, path string, body json.RawMessage) (json.RawMessage, *ResultInfo, error) {
	var requestBody any
	if len(body) > 0 {
		requestBody = body
	}
	return c.do(ctx, method, path, requestBody)
}

type RequestPreview struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    json.RawMessage   `json:"body,omitempty"`
}

func (c *Client) PreviewRequest(method, path string, body json.RawMessage) RequestPreview {
	return RequestPreview{
		Method: method,
		URL:    c.baseURL + path,
		Headers: map[string]string{
			"Authorization": redactToken(c.token),
			"Content-Type":  "application/json",
		},
		Body: body,
	}
}

func (c *Client) VerifyToken(ctx context.Context) (json.RawMessage, error) {
	raw, _, err := c.Get(ctx, "/user/tokens/verify", nil)
	return raw, err
}

func (c *Client) Accounts(ctx context.Context, params url.Values) ([]json.RawMessage, *ResultInfo, error) {
	raw, info, err := c.Get(ctx, "/accounts", params)
	if err != nil {
		return nil, nil, err
	}
	items, err := DecodeResultList(raw)
	return items, info, err
}

func (c *Client) Zones(ctx context.Context, params url.Values) ([]json.RawMessage, *ResultInfo, error) {
	raw, info, err := c.Get(ctx, "/zones", params)
	if err != nil {
		return nil, nil, err
	}
	items, err := DecodeResultList(raw)
	return items, info, err
}

func (c *Client) Zone(ctx context.Context, zoneID string) (json.RawMessage, error) {
	raw, _, err := c.Get(ctx, "/zones/"+url.PathEscape(zoneID), nil)
	return raw, err
}

func (c *Client) DNSRecords(ctx context.Context, zoneID string, params url.Values) ([]json.RawMessage, *ResultInfo, error) {
	raw, info, err := c.Get(ctx, "/zones/"+url.PathEscape(zoneID)+"/dns_records", params)
	if err != nil {
		return nil, nil, err
	}
	items, err := DecodeResultList(raw)
	return items, info, err
}

func (c *Client) ZoneSetting(ctx context.Context, zoneID, settingID string) (json.RawMessage, error) {
	raw, _, err := c.Get(ctx, "/zones/"+url.PathEscape(zoneID)+"/settings/"+url.PathEscape(settingID), nil)
	return raw, err
}

func (c *Client) Rulesets(ctx context.Context, scope, scopeID string, params url.Values) ([]json.RawMessage, *ResultInfo, error) {
	raw, info, err := c.Get(ctx, "/"+url.PathEscape(scope)+"/"+url.PathEscape(scopeID)+"/rulesets", params)
	if err != nil {
		return nil, nil, err
	}
	items, err := DecodeResultList(raw)
	return items, info, err
}

func (c *Client) CacheSetting(ctx context.Context, zoneID, settingPath string) (json.RawMessage, error) {
	raw, _, err := c.Get(ctx, "/zones/"+url.PathEscape(zoneID)+"/cache/"+settingPath, nil)
	return raw, err
}

func (c *Client) WaitingRooms(ctx context.Context, scope, scopeID string, params url.Values) ([]json.RawMessage, *ResultInfo, error) {
	raw, info, err := c.Get(ctx, "/"+url.PathEscape(scope)+"/"+url.PathEscape(scopeID)+"/waiting_rooms", params)
	if err != nil {
		return nil, nil, err
	}
	items, err := DecodeResultList(raw)
	return items, info, err
}

func (c *Client) WaitingRoom(ctx context.Context, zoneID, roomID string) (json.RawMessage, error) {
	raw, _, err := c.Get(ctx, "/zones/"+url.PathEscape(zoneID)+"/waiting_rooms/"+url.PathEscape(roomID), nil)
	return raw, err
}

func (c *Client) do(ctx context.Context, method, path string, body any) (json.RawMessage, *ResultInfo, error) {
	req, err := c.buildRequest(ctx, method, path, body)
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, agenterrors.Wrap(err, agenterrors.FixableByRetry).WithHint("Network error: check connectivity and retry")
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, agenterrors.Wrap(err, agenterrors.FixableByRetry)
	}
	if c.debug {
		w := output.NewNDJSONWriter(output.Stderr())
		_ = w.WriteMetaLine(output.MetaKeyRequest, map[string]any{
			"method": req.Method,
			"url":    req.URL.String(),
			"status": resp.StatusCode,
		})
	}
	if resp.StatusCode >= 400 {
		return nil, nil, classifyHTTPError(resp.StatusCode, respBody)
	}
	result, info, err := DecodeEnvelope(respBody)
	if err != nil {
		return nil, nil, err
	}
	return result, info, nil
}

func (c *Client) buildRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func buildPath(base string, params url.Values) string {
	if len(params) == 0 {
		return base
	}
	if encoded := params.Encode(); encoded != "" {
		return base + "?" + encoded
	}
	return base
}

func redactToken(token string) string {
	if token == "" {
		return "(unset)"
	}
	return fmt.Sprintf("Bearer (set, %d chars)", len(token))
}
