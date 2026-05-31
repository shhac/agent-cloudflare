package mockcloudflare

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerRequiresBearerToken(t *testing.T) {
	server := httptest.NewServer(NewServer())
	defer server.Close()

	resp, err := http.Get(server.URL + "/zones")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}
