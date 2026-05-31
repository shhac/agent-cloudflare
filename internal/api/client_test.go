package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
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
}
