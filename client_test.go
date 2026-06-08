package ecosystems

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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
func TestNewClientDefaultBatchSize(t *testing.T) {
	client, err := NewClient("test-agent/1.0")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.batchSize != MaxBulkLookupSize {
		t.Errorf("batchSize = %d, want %d", client.batchSize, MaxBulkLookupSize)
	}
}

func TestWithBatchSize(t *testing.T) {
	tests := []struct {
		name string
		size int
		want int
	}{
		{"valid", 25, 25},
		{"zero falls back to max", 0, MaxBulkLookupSize},
		{"negative falls back to max", -5, MaxBulkLookupSize},
		{"over max falls back to max", MaxBulkLookupSize + 50, MaxBulkLookupSize},
		{"at max stays at max", MaxBulkLookupSize, MaxBulkLookupSize},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient("test-agent/1.0", WithBatchSize(tt.size))
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}
			if client.batchSize != tt.want {
				t.Errorf("batchSize = %d, want %d", client.batchSize, tt.want)
			}
		})
	}
}

func TestBulkLookupBatching(t *testing.T) {
	var mu sync.Mutex
	var batches [][]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Purls []string `json:"purls"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		batches = append(batches, body.Purls)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	client, err := NewClient("test-agent/1.0",
		WithPackagesServer(srv.URL),
		WithBatchSize(3),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	purls := []string{"pkg:a/1", "pkg:a/2", "pkg:a/3", "pkg:a/4", "pkg:a/5", "pkg:a/6", "pkg:a/7"}
	if _, err := client.BulkLookup(context.Background(), purls); err != nil {
		t.Fatalf("BulkLookup() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(batches) != 3 {
		t.Fatalf("got %d batches, want 3", len(batches))
	}
	wantSizes := []int{3, 3, 1}
	for i, b := range batches {
		if len(b) != wantSizes[i] {
			t.Errorf("batch %d size = %d, want %d", i, len(b), wantSizes[i])
		}
	}
}
