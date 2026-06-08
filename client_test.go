package ecosystems

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestLooksLikeStreamErr(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("connection refused"), false},
		{errors.New("stream error: stream ID 1; INTERNAL_ERROR; received from peer"), true},
		{errors.New("http2: server sent GOAWAY"), true},
		{errors.New("ENHANCE_YOUR_CALM"), true},
	}
	for _, tt := range tests {
		if got := looksLikeStreamErr(tt.err); got != tt.want {
			t.Errorf("looksLikeStreamErr(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

func TestBulkLookupHintOn429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client, err := NewClient("test-agent/1.0", WithPackagesServer(srv.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.BulkLookup(context.Background(), []string{"pkg:npm/lodash"})
	if err == nil {
		t.Fatal("BulkLookup() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error = %q, want it to mention status 429", err.Error())
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("error = %q, want rate-limit hint", err.Error())
	}
}

func TestBulkLookupNoHintOn5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client, err := NewClient("test-agent/1.0", WithPackagesServer(srv.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.BulkLookup(context.Background(), []string{"pkg:npm/lodash"})
	if err == nil {
		t.Fatal("BulkLookup() expected error, got nil")
	}
	if strings.Contains(err.Error(), "rate limited") {
		t.Errorf("error = %q, 500 is not rate-limiting", err.Error())
	}
}

func TestBulkLookupNoHintOnPlain400(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid purl"}`))
	}))
	defer srv.Close()

	client, err := NewClient("test-agent/1.0", WithPackagesServer(srv.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.BulkLookup(context.Background(), []string{"not-a-purl"})
	if err == nil {
		t.Fatal("BulkLookup() expected error, got nil")
	}
	if strings.Contains(err.Error(), "rate limited") {
		t.Errorf("error = %q, did not expect rate-limit hint on plain 400", err.Error())
	}
	if !strings.Contains(err.Error(), "invalid purl") {
		t.Errorf("error = %q, want it to preserve the 400 detail", err.Error())
	}
}
