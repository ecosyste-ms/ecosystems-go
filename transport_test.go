package ecosystems_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	ecosystems "github.com/ecosyste-ms/ecosystems-go"
)

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestNewClientWithHTTPClient(t *testing.T) {
	const userAgent = "custom-transport-test"
	calls := 0
	transport := testTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/registries" {
			t.Errorf("request = %s %s, want GET /api/v1/registries", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("User-Agent"); got != userAgent {
			t.Errorf("User-Agent = %q, want %q", got, userAgent)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`[{"name":"npmjs.org"}]`)),
			Request:    r,
		}, nil
	})
	client, err := ecosystems.NewClient(userAgent,
		ecosystems.WithHTTPClient(&http.Client{Transport: transport}),
	)
	if err != nil {
		t.Fatal(err)
	}
	registries, err := client.ListRegistries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("requests = %d, want 1", calls)
	}
	if len(registries) != 1 || registries[0].Name != "npmjs.org" {
		t.Fatalf("registries = %+v, want npmjs.org", registries)
	}
}
