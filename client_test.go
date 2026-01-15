package ecosystems

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient("test-agent/1.0")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
}

func TestNewClientRequiresUserAgent(t *testing.T) {
	_, err := NewClient("")
	if err == nil {
		t.Fatal("NewClient() with empty userAgent should error")
	}
}

func TestNewClientWithOptions(t *testing.T) {
	client, err := NewClient("test-agent/1.0",
		WithPackagesServer("https://custom.packages.server"),
		WithReposServer("https://custom.repos.server"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.userAgent != "test-agent/1.0" {
		t.Errorf("userAgent = %q, want %q", client.userAgent, "test-agent/1.0")
	}
}

func TestBulkLookupEmpty(t *testing.T) {
	client, err := NewClient("test-agent/1.0")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	results, err := client.BulkLookup(context.Background(), []string{})
	if err != nil {
		t.Fatalf("BulkLookup() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("BulkLookup([]) = %d results, want 0", len(results))
	}
}
