package ecosystems_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	ecosystems "github.com/ecosyste-ms/ecosystems-go"
	packageurl "github.com/package-url/packageurl-go"
)

func TestClientWithUpstreamPURL(t *testing.T) {
	tests := []struct {
		input    string
		registry string
		name     string
	}{
		{"pkg:npm/%40babel/core@7.20.0", "npmjs.org", "@babel/core"},
		{"pkg:maven/org.apache.commons/commons-lang3@3.12.0", "repo1.maven.org", "org.apache.commons:commons-lang3"},
		{"pkg:apk/alpine/curl@8.0.1", "alpine-edge", "curl"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			testClientWithUpstreamPURL(t, tt.input, tt.registry, tt.name)
		})
	}
}

func testClientWithUpstreamPURL(t *testing.T, input, registry, name string) {
	t.Helper()
	p, err := packageurl.FromString(input)
	if err != nil {
		t.Fatal(err)
	}
	var parsed packageurl.PackageURL
	parsed, err = ecosystems.ParsePURL(strings.TrimPrefix(input, "pkg:"))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.String() != p.String() {
		t.Fatalf("ParsePURL() = %q, want %q", parsed.String(), p.String())
	}

	basePath := "/api/v1/registries/" + registry + "/packages/" + url.PathEscape(name)
	requests := make(map[string]int)
	transport := testTransport(func(r *http.Request) (*http.Response, error) {
		path := r.URL.EscapedPath()
		requests[path]++
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		var body string
		switch path {
		case basePath:
			body = fmt.Sprintf(`{"name":%q}`, name)
		case basePath + "/versions/" + p.Version:
			body = fmt.Sprintf(`{"number":%q}`, p.Version)
		case basePath + "/versions":
			body = fmt.Sprintf(`[{"number":%q}]`, p.Version)
		default:
			t.Errorf("unexpected request path: %s", path)
			body = `{}`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})
	client, err := ecosystems.NewClient("purl-test", ecosystems.WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := client.LookupPURL(context.Background(), p)
	if err != nil || pkg == nil || pkg.Name != name {
		t.Fatalf("LookupPURL() = %+v, %v; want %s", pkg, err, name)
	}
	version, err := client.GetVersionPURL(context.Background(), parsed)
	if err != nil || version == nil || version.Number != p.Version {
		t.Fatalf("GetVersionPURL() = %+v, %v; want %s", version, err, p.Version)
	}
	versions, err := client.GetAllVersionsPURL(context.Background(), parsed)
	if err != nil || len(versions) != 1 || versions[0].Number != p.Version {
		t.Fatalf("GetAllVersionsPURL() = %+v, %v; want %s", versions, err, p.Version)
	}
	for _, path := range []string{basePath, basePath + "/versions/" + p.Version, basePath + "/versions"} {
		if requests[path] != 1 {
			t.Errorf("requests to %s = %d, want 1", path, requests[path])
		}
	}
}
