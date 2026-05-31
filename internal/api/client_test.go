package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	agenterrors "github.com/shhac/agent-cloudflare/internal/errors"
)

func TestGetSendsBearerTokenAndQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer cfut_test" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := r.URL.Query().Get("name"); got != "example.com" {
			t.Fatalf("name = %q", got)
		}
		_, _ = w.Write([]byte(`{"success":true,"result":[{"id":"zone_1"}],"errors":[],"messages":[],"result_info":{"page":1,"per_page":20,"count":1,"total_count":1,"total_pages":1}}`))
	}))
	defer server.Close()

	client := NewClient(Options{Token: "cfut_test", BaseURL: server.URL})
	items, info, err := client.Zones(t.Context(), url.Values{"name": []string{"example.com"}})
	if err != nil {
		t.Fatalf("Zones() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if info == nil || info.TotalCount != 1 {
		t.Fatalf("info = %#v, want total count", info)
	}
}

func TestClassifyHTTPErrorUsesCloudflareEnvelope(t *testing.T) {
	err := classifyHTTPError(403, []byte(`{"success":false,"errors":[{"code":9109,"message":"Invalid access token"}]}`))
	if err.FixableBy != "human" {
		t.Fatalf("FixableBy = %q, want human", err.FixableBy)
	}
	if err.Hint == "" {
		t.Fatalf("Hint should be populated")
	}
	if err.Message == "" || err.Message == "HTTP 403" {
		t.Fatalf("Message = %q, want Cloudflare message", err.Message)
	}
}

func TestClassifyHTTPErrorProvidesActionableHints(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       []byte
		fixableBy  agenterrors.FixableBy
		hintNeedle string
	}{
		{name: "auth", status: 401, body: []byte(`{"errors":[{"code":10000,"message":"auth failed"}]}`), fixableBy: agenterrors.FixableByHuman, hintNeedle: "profiles check"},
		{name: "permission", status: 403, body: []byte(`{"errors":[{"code":10001,"message":"forbidden"}]}`), fixableBy: agenterrors.FixableByHuman, hintNeedle: "permission groups"},
		{name: "not found", status: 404, body: []byte(`{"errors":[{"code":7003,"message":"not found"}]}`), fixableBy: agenterrors.FixableByAgent, hintNeedle: "rediscover"},
		{name: "rate limit", status: 429, body: []byte(`{"errors":[{"message":"too many"}]}`), fixableBy: agenterrors.FixableByRetry, hintNeedle: "smaller time window"},
		{name: "server", status: 500, body: []byte(`{"errors":[{"message":"server"}]}`), fixableBy: agenterrors.FixableByRetry, hintNeedle: "--debug"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifyHTTPError(tt.status, tt.body)
			if err.FixableBy != tt.fixableBy {
				t.Fatalf("FixableBy = %q, want %q", err.FixableBy, tt.fixableBy)
			}
			if !strings.Contains(err.Hint, tt.hintNeedle) {
				t.Fatalf("Hint = %q, want %q", err.Hint, tt.hintNeedle)
			}
		})
	}
}

func TestGraphQLReturnsHumanFixableHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":null,"errors":[{"message":"permission denied","extensions":{"code":"authz"}}]}`))
	}))
	defer server.Close()

	client := NewClient(Options{Token: "cfut_test", BaseURL: server.URL})
	_, err := client.GraphQL(t.Context(), "query { viewer { zones { zoneTag } } }", nil)
	var apiErr *agenterrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %#v, want APIError", err)
	}
	if apiErr.FixableBy != agenterrors.FixableByHuman {
		t.Fatalf("FixableBy = %q, want human", apiErr.FixableBy)
	}
	if !strings.Contains(apiErr.Hint, "Analytics Read") {
		t.Fatalf("Hint = %q, want Analytics Read", apiErr.Hint)
	}
}
