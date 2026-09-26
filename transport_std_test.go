//go:build !tinygo

package ecosystems

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientHTTP2(t *testing.T) {
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 {
			t.Errorf("protocol = %s, want HTTP/2", r.Proto)
		}
		if r.URL.Path != "/registries" {
			t.Errorf("path = %q, want /registries", r.URL.Path)
		}
		writeTestJSON(w, `[{"name":"npmjs.org"}]`)
	}))
	srv.EnableHTTP2 = true
	srv.StartTLS()
	defer srv.Close()

	client, err := NewClient("http2-test", WithPackagesServer(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	retry, ok := client.httpClient.Transport.(*retryRoundTripper)
	if !ok {
		t.Fatalf("transport = %T, want retryRoundTripper", client.httpClient.Transport)
	}
	transport, ok := retry.base.(*http.Transport)
	if !ok {
		t.Fatalf("base transport = %T, want http.Transport", retry.base)
	}
	transport.TLSClientConfig = srv.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	defer transport.CloseIdleConnections()

	registries, err := client.ListRegistries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(registries) != 1 || registries[0].Name != "npmjs.org" {
		t.Fatalf("registries = %+v, want npmjs.org", registries)
	}
}
