package ecosystems

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLookupPackagesByRepositoryURLFollowsLinkAndCaps(t *testing.T) {
	var sawNextUserAgent bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/packages/lookup" {
			t.Fatalf("path = %q, want /packages/lookup", r.URL.Path)
		}
		if r.URL.Query().Get("page") == "2" {
			sawNextUserAgent = r.Header.Get("User-Agent") == "test-agent/1.0"
			writeTestJSON(w, `[
				{"purl":"pkg:npm/b","name":"b","ecosystem":"npm"},
				{"purl":"pkg:npm/c","name":"c","ecosystem":"npm"}
			]`)
			return
		}
		if got := r.URL.Query().Get("repository_url"); got != "https://github.com/acme/widget" {
			t.Fatalf("repository_url = %q", got)
		}
		w.Header().Set("Link", fmt.Sprintf(`<%s/packages/lookup?page=2>; rel="next"`, "http://"+r.Host))
		writeTestJSON(w, `[{"purl":"pkg:npm/a","name":"a","ecosystem":"npm"}]`)
	}))
	defer srv.Close()

	client, err := NewClient("test-agent/1.0", WithPackagesServer(srv.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	got, err := client.LookupPackagesByRepositoryURL(context.Background(), "https://github.com/acme/widget", 2)
	if err != nil {
		t.Fatalf("LookupPackagesByRepositoryURL() error = %v", err)
	}
	if len(got) != 2 || got[0].PURL != "pkg:npm/a" || got[1].PURL != "pkg:npm/b" {
		t.Fatalf("packages = %+v, want first two linked rows", got)
	}
	if !sawNextUserAgent {
		t.Fatal("linked request did not carry configured User-Agent")
	}
}

func TestLookupPackagesByRepositoryURLSignalsPageCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next := r.URL.Query().Get("page")
		if next == "" {
			next = "1"
		}
		w.Header().Set("Link", fmt.Sprintf(`<%s/packages/lookup?page=%s>; rel="next"`, "http://"+r.Host, next+"x"))
		writeTestJSON(w, `[{"purl":"pkg:npm/a","name":"a","ecosystem":"npm"}]`)
	}))
	defer srv.Close()

	client, err := NewClient("test-agent/1.0", WithPackagesServer(srv.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, err = client.LookupPackagesByRepositoryURL(context.Background(), "https://github.com/acme/widget", 0)
	if err == nil || !strings.Contains(err.Error(), "pagination exceeded max pages") {
		t.Fatalf("error = %v, want pagination cap error", err)
	}
}

func TestLookupPackagesByPURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("purl"); got != "pkg:gem/rake" {
			t.Fatalf("purl = %q", got)
		}
		writeTestJSON(w, `[{"purl":"pkg:gem/rake","name":"rake","ecosystem":"rubygems","repository_url":"https://github.com/ruby/rake"}]`)
	}))
	defer srv.Close()

	client, err := NewClient("test-agent/1.0", WithPackagesServer(srv.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	got, err := client.LookupPackagesByPURL(context.Background(), "pkg:gem/rake")
	if err != nil {
		t.Fatalf("LookupPackagesByPURL() error = %v", err)
	}
	if len(got) != 1 || !got[0].RepositoryURL.IsSpecified() || got[0].RepositoryURL.MustGet() != "https://github.com/ruby/rake" {
		t.Fatalf("packages = %+v", got)
	}
}

func TestGetDependentPackagesFollowsLink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/registries/npmjs.org/packages/lodash/dependent_packages" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("page") == "2" {
			writeTestJSON(w, `[{"purl":"pkg:npm/downstream-b","name":"downstream-b","ecosystem":"npm"}]`)
			return
		}
		w.Header().Set("Link", fmt.Sprintf(`<%s/registries/npmjs.org/packages/lodash/dependent_packages?page=2>; rel=next`, "http://"+r.Host))
		writeTestJSON(w, `[{"purl":"pkg:npm/downstream-a","name":"downstream-a","ecosystem":"npm"}]`)
	}))
	defer srv.Close()

	client, err := NewClient("test-agent/1.0", WithPackagesServer(srv.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	got, err := client.GetDependentPackages(context.Background(), "npmjs.org", "lodash", 0)
	if err != nil {
		t.Fatalf("GetDependentPackages() error = %v", err)
	}
	if len(got) != 2 || got[1].PURL != "pkg:npm/downstream-b" {
		t.Fatalf("dependents = %+v", got)
	}
}

func TestGetAdvisoriesByRepoURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("repository_url"); got != "https://github.com/rails/rails" {
			t.Fatalf("repository_url = %q", got)
		}
		writeTestJSON(w, `[{"uuid":"GHSA-1","repository_url":"https://github.com/rails/rails"}]`)
	}))
	defer srv.Close()

	client, err := NewClient("test-agent/1.0", WithAdvisoriesServer(srv.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	got, err := client.GetAdvisoriesByRepoURL(context.Background(), "https://github.com/rails/rails", 10)
	if err != nil {
		t.Fatalf("GetAdvisoriesByRepoURL() error = %v", err)
	}
	if len(got) != 1 || got[0].UUID == nil || *got[0].UUID != "GHSA-1" {
		t.Fatalf("advisories = %+v", got)
	}
}

func TestGetCommitsAndIssuesSummary(t *testing.T) {
	commitsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(w, `{"full_name":"acme/widget","total_commits":42}`)
	}))
	defer commitsSrv.Close()
	issuesSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(w, `{"full_name":"acme/widget","issues_count":7}`)
	}))
	defer issuesSrv.Close()

	client, err := NewClient("test-agent/1.0",
		WithCommitsServer(commitsSrv.URL),
		WithIssuesServer(issuesSrv.URL),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	commits, err := client.GetCommitsSummary(context.Background(), "https://github.com/acme/widget")
	if err != nil {
		t.Fatalf("GetCommitsSummary() error = %v", err)
	}
	if commits.TotalCommits == nil || *commits.TotalCommits != 42 {
		t.Fatalf("commits summary = %+v", commits)
	}
	issues, err := client.GetIssuesSummary(context.Background(), "https://github.com/acme/widget")
	if err != nil {
		t.Fatalf("GetIssuesSummary() error = %v", err)
	}
	if issues.IssuesCount == nil || *issues.IssuesCount != 7 {
		t.Fatalf("issues summary = %+v", issues)
	}
}

func TestDefaultHTTPClientRetriesGET(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		if hits == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		writeTestJSON(w, `[]`)
	}))
	defer srv.Close()

	client, err := NewClient("test-agent/1.0", WithPackagesServer(srv.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if _, err := client.ListRegistries(context.Background()); err != nil {
		t.Fatalf("ListRegistries() error = %v", err)
	}
	if hits != 2 {
		t.Fatalf("hits = %d, want retry", hits)
	}
}

func TestNextLink(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{`<https://x/api?page=2>; rel="next"`, "https://x/api?page=2"},
		{`<https://x/api?page=3>; rel="last", <https://x/api?page=2>; rel="next"`, "https://x/api?page=2"},
		{`<https://x/api?page=2>; rel=next`, "https://x/api?page=2"},
		{`<https://x/api?page=9>; rel="last"`, ""},
		{`<https://x/api?page=2&f=a,b,c>; rel="next"`, "https://x/api?page=2&f=a,b,c"},
		{"", ""},
		{"garbage", ""},
		{`<https://x/api?page=2; rel="next"`, ""},
	}
	for _, tt := range cases {
		if got := nextLink(tt.header); got != tt.want {
			t.Errorf("nextLink(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestLookupWrapperReportsStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client, err := NewClient("test-agent/1.0", WithPackagesServer(srv.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, err = client.LookupPackagesByPURL(context.Background(), "pkg:npm/lodash")
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("error = %v, want status 500", err)
	}
}

func writeTestJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}
