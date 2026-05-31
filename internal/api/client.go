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

func (c *Client) Post(ctx context.Context, path string, body any) (json.RawMessage, *ResultInfo, error) {
	return c.do(ctx, http.MethodPost, path, body)
}

func (c *Client) Patch(ctx context.Context, path string, body any) (json.RawMessage, *ResultInfo, error) {
	return c.do(ctx, http.MethodPatch, path, body)
}

func (c *Client) RawRequest(ctx context.Context, method, path string, body json.RawMessage) (json.RawMessage, *ResultInfo, error) {
	var requestBody any
	if len(body) > 0 {
		requestBody = body
	}
	return c.do(ctx, method, path, requestBody)
}

func (c *Client) GraphQL(ctx context.Context, query string, variables map[string]any) (json.RawMessage, error) {
	body := map[string]any{
		"query":     query,
		"variables": variables,
	}
	resp, err := c.sendRaw(ctx, http.MethodPost, "/graphql", body)
	if err != nil {
		return nil, err
	}
	if resp.status >= 400 {
		return nil, classifyHTTPError(resp.status, resp.body)
	}
	return json.RawMessage(resp.body), nil
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

func (c *Client) CreateDNSRecord(ctx context.Context, zoneID string, body map[string]any) (json.RawMessage, error) {
	raw, _, err := c.Post(ctx, "/zones/"+url.PathEscape(zoneID)+"/dns_records", body)
	return raw, err
}

func (c *Client) UpdateDNSRecord(ctx context.Context, zoneID, recordID string, body map[string]any) (json.RawMessage, error) {
	raw, _, err := c.Patch(ctx, "/zones/"+url.PathEscape(zoneID)+"/dns_records/"+url.PathEscape(recordID), body)
	return raw, err
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

func (c *Client) PurgeCache(ctx context.Context, zoneID string, body map[string]any) (json.RawMessage, error) {
	raw, _, err := c.Post(ctx, "/zones/"+url.PathEscape(zoneID)+"/purge_cache", body)
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

func (c *Client) UpdateWaitingRoom(ctx context.Context, zoneID, roomID string, body map[string]any) (json.RawMessage, error) {
	raw, _, err := c.Patch(ctx, "/zones/"+url.PathEscape(zoneID)+"/waiting_rooms/"+url.PathEscape(roomID), body)
	return raw, err
}

func (c *Client) Workers(ctx context.Context, accountID string, params url.Values) ([]json.RawMessage, *ResultInfo, error) {
	raw, info, err := c.Get(ctx, "/accounts/"+url.PathEscape(accountID)+"/workers/scripts", params)
	if err != nil {
		return nil, nil, err
	}
	items, err := DecodeResultList(raw)
	return items, info, err
}

func (c *Client) WorkerSubdomain(ctx context.Context, accountID, scriptName string) (json.RawMessage, error) {
	raw, _, err := c.Get(ctx, "/accounts/"+url.PathEscape(accountID)+"/workers/scripts/"+url.PathEscape(scriptName)+"/subdomain", nil)
	return raw, err
}

func (c *Client) WorkerVersions(ctx context.Context, accountID, scriptName string, params url.Values) ([]json.RawMessage, *ResultInfo, error) {
	raw, info, err := c.Get(ctx, "/accounts/"+url.PathEscape(accountID)+"/workers/scripts/"+url.PathEscape(scriptName)+"/versions", params)
	if err != nil {
		return nil, nil, err
	}
	items, err := DecodeResultList(raw)
	return items, info, err
}

func (c *Client) KVNamespaces(ctx context.Context, accountID string, params url.Values) ([]json.RawMessage, *ResultInfo, error) {
	raw, info, err := c.Get(ctx, "/accounts/"+url.PathEscape(accountID)+"/storage/kv/namespaces", params)
	if err != nil {
		return nil, nil, err
	}
	items, err := DecodeResultList(raw)
	return items, info, err
}

func (c *Client) KVNamespace(ctx context.Context, accountID, namespaceID string) (json.RawMessage, error) {
	raw, _, err := c.Get(ctx, "/accounts/"+url.PathEscape(accountID)+"/storage/kv/namespaces/"+url.PathEscape(namespaceID), nil)
	return raw, err
}

func (c *Client) R2Buckets(ctx context.Context, accountID string, params url.Values) ([]json.RawMessage, *ResultInfo, error) {
	raw, info, err := c.Get(ctx, "/accounts/"+url.PathEscape(accountID)+"/r2/buckets", params)
	if err != nil {
		return nil, nil, err
	}
	items, err := DecodeBucketList(raw)
	return items, info, err
}

func (c *Client) R2Bucket(ctx context.Context, accountID, bucketName string) (json.RawMessage, error) {
	raw, _, err := c.Get(ctx, "/accounts/"+url.PathEscape(accountID)+"/r2/buckets/"+url.PathEscape(bucketName), nil)
	return raw, err
}

func (c *Client) AuditLogs(ctx context.Context, accountID string, params url.Values) ([]json.RawMessage, *ResultInfo, error) {
	raw, info, err := c.Get(ctx, "/accounts/"+url.PathEscape(accountID)+"/logs/audit", params)
	if err != nil {
		return nil, nil, err
	}
	items, err := DecodeResultList(raw)
	return items, info, err
}

func (c *Client) do(ctx context.Context, method, path string, body any) (json.RawMessage, *ResultInfo, error) {
	resp, err := c.sendRaw(ctx, method, path, body)
	if err != nil {
		return nil, nil, err
	}
	if resp.status >= 400 {
		return nil, nil, classifyHTTPError(resp.status, resp.body)
	}
	result, info, err := DecodeEnvelope(resp.body)
	if err != nil {
		return nil, nil, err
	}
	return result, info, nil
}

type rawResponse struct {
	status int
	body   []byte
}

func (c *Client) sendRaw(ctx context.Context, method, path string, body any) (*rawResponse, error) {
	req, err := c.buildRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByRetry).WithHint("Network error: check connectivity and retry")
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByRetry)
	}
	if c.debug {
		w := output.NewNDJSONWriter(output.Stderr())
		_ = w.WriteMetaLine(output.MetaKeyRequest, map[string]any{
			"method": req.Method,
			"url":    req.URL.String(),
			"status": resp.StatusCode,
		})
	}
	return &rawResponse{status: resp.StatusCode, body: respBody}, nil
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
