//go:build integration

package ecosystems

import (
	"context"
	"testing"
	"time"
)

func TestIntegrationBulkLookup(t *testing.T) {
	client, err := NewClient("ecosystems-go-test/1.0")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	purls := []string{
		"pkg:gem/rails",
		"pkg:npm/lodash",
		"pkg:pypi/requests",
	}

	results, err := client.BulkLookup(ctx, purls)
	if err != nil {
		t.Fatalf("BulkLookup() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("BulkLookup() returned no results")
	}

	// Check that we got at least rails
	if pkg, ok := results["pkg:gem/rails"]; ok {
		if pkg.Name != "rails" {
			t.Errorf("rails package name = %q, want %q", pkg.Name, "rails")
		}
		// Ecosystem is "rubygems", registry is "rubygems.org"
		if pkg.Ecosystem != "rubygems" {
			t.Errorf("rails ecosystem = %q, want %q", pkg.Ecosystem, "rubygems")
		}
	} else {
		t.Error("BulkLookup() missing rails result")
	}
}

func TestIntegrationLookup(t *testing.T) {
	client, err := NewClient("ecosystems-go-test/1.0")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pkg, err := client.Lookup(ctx, "pkg:gem/rake")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}

	if pkg == nil {
		t.Fatal("Lookup() returned nil")
	}

	if pkg.Name != "rake" {
		t.Errorf("Name = %q, want %q", pkg.Name, "rake")
	}
}

func TestIntegrationGetVersion(t *testing.T) {
	client, err := NewClient("ecosystems-go-test/1.0")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	version, err := client.GetVersion(ctx, "rubygems.org", "rake", "13.0.0")
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}

	if version == nil {
		t.Fatal("GetVersion() returned nil")
	}

	if version.Number != "13.0.0" {
		t.Errorf("Number = %q, want %q", version.Number, "13.0.0")
	}
}

func TestIntegrationGetAllVersions(t *testing.T) {
	client, err := NewClient("ecosystems-go-test/1.0")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	versions, err := client.GetAllVersions(ctx, "rubygems.org", "rake")
	if err != nil {
		t.Fatalf("GetAllVersions() error = %v", err)
	}

	if len(versions) == 0 {
		t.Error("GetAllVersions() returned no versions")
	}

	// Rake has many versions
	if len(versions) < 10 {
		t.Errorf("GetAllVersions() returned %d versions, expected more", len(versions))
	}
}

func TestIntegrationLookupPURL(t *testing.T) {
	client, err := NewClient("ecosystems-go-test/1.0")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	purl, err := ParsePURL("pkg:npm/lodash")
	if err != nil {
		t.Fatalf("ParsePURL() error = %v", err)
	}

	pkg, err := client.LookupPURL(ctx, purl)
	if err != nil {
		t.Fatalf("LookupPURL() error = %v", err)
	}

	if pkg == nil {
		t.Fatal("LookupPURL() returned nil")
	}

	if pkg.Name != "lodash" {
		t.Errorf("Name = %q, want %q", pkg.Name, "lodash")
	}
}

func TestIntegrationGetVersionPURL(t *testing.T) {
	client, err := NewClient("ecosystems-go-test/1.0")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	purl, err := ParsePURL("pkg:npm/lodash@4.17.21")
	if err != nil {
		t.Fatalf("ParsePURL() error = %v", err)
	}

	version, err := client.GetVersionPURL(ctx, purl)
	if err != nil {
		t.Fatalf("GetVersionPURL() error = %v", err)
	}

	if version == nil {
		t.Fatal("GetVersionPURL() returned nil")
	}

	if version.Number != "4.17.21" {
		t.Errorf("Number = %q, want %q", version.Number, "4.17.21")
	}
}

func TestIntegrationListRegistries(t *testing.T) {
	client, err := NewClient("ecosystems-go-test/1.0")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	registries, err := client.ListRegistries(ctx)
	if err != nil {
		t.Fatalf("ListRegistries() error = %v", err)
	}

	if len(registries) == 0 {
		t.Error("ListRegistries() returned no registries")
	}

	// Find rubygems.org
	found := false
	for _, r := range registries {
		if r.Name == "rubygems.org" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ListRegistries() missing rubygems.org")
	}
}
