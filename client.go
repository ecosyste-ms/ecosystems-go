// Package ecosystems provides a client for the ecosyste.ms APIs.
//
// This package wraps the generated OpenAPI clients for packages.ecosyste.ms
// and repos.ecosyste.ms, providing a higher-level API for common operations.
package ecosystems

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/ecosyste-ms/ecosystems-go/packages"
	"github.com/ecosyste-ms/ecosystems-go/repos"
)

const (
	DefaultPackagesServer = "https://packages.ecosyste.ms/api/v1"
	DefaultReposServer    = "https://repos.ecosyste.ms/api/v1"
	DefaultTimeout        = 30 * time.Second
	MaxBulkLookupSize     = 100
)

type Client struct {
	packagesClient *packages.ClientWithResponses
	reposClient    *repos.ClientWithResponses
	userAgent      string
}

type Option func(*clientConfig)

type clientConfig struct {
	packagesServer string
	reposServer    string
	httpClient     *http.Client
	userAgent      string
	fromEmail      string
	apiKey         string
}

func WithPackagesServer(server string) Option {
	return func(c *clientConfig) {
		c.packagesServer = server
	}
}

func WithReposServer(server string) Option {
	return func(c *clientConfig) {
		c.reposServer = server
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *clientConfig) {
		c.httpClient = client
	}
}

// WithFrom sets the From header (email address) for API requests.
// This helps ecosyste.ms identify who is making requests.
func WithFrom(email string) Option {
	return func(c *clientConfig) {
		c.fromEmail = email
	}
}

// WithAPIKey sets the API key for authenticated requests.
// This provides higher rate limits and access to additional features.
func WithAPIKey(key string) Option {
	return func(c *clientConfig) {
		c.apiKey = key
	}
}

// defaultHTTPClient creates an optimized HTTP client for the ecosyste.ms APIs.
// Features:
//   - HTTP/2 enabled (automatic over HTTPS)
//   - Connection keep-alive with pooling
//   - Gzip compression (Accept-Encoding handled by transport)
func defaultHTTPClient() *http.Client {
	transport := &http.Transport{
		// Connection pooling
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     90 * time.Second,

		// Timeouts
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		// Enable compression (gzip)
		DisableCompression: false,

		// HTTP/2 is enabled by default for HTTPS when using http.Transport
		ForceAttemptHTTP2: true,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   DefaultTimeout,
	}
}

// NewClient creates a new ecosyste.ms API client.
// The userAgent parameter is required and should identify your application.
func NewClient(userAgent string, opts ...Option) (*Client, error) {
	if userAgent == "" {
		return nil, fmt.Errorf("userAgent is required")
	}

	cfg := &clientConfig{
		packagesServer: DefaultPackagesServer,
		reposServer:    DefaultReposServer,
		httpClient:     defaultHTTPClient(),
		userAgent:      userAgent,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Note: Don't set Accept-Encoding manually - the Transport handles gzip
	// automatically when DisableCompression is false (the default).
	// Setting it manually disables automatic decompression.
	addHeaders := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("User-Agent", cfg.userAgent)
		if cfg.fromEmail != "" {
			req.Header.Set("From", cfg.fromEmail)
		}
		if cfg.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.apiKey)
		}
		return nil
	}

	pkgClient, err := packages.NewClientWithResponses(
		cfg.packagesServer,
		packages.WithHTTPClient(cfg.httpClient),
		packages.WithRequestEditorFn(addHeaders),
	)
	if err != nil {
		return nil, fmt.Errorf("creating packages client: %w", err)
	}

	repoClient, err := repos.NewClientWithResponses(
		cfg.reposServer,
		repos.WithHTTPClient(cfg.httpClient),
		repos.WithRequestEditorFn(addHeaders),
	)
	if err != nil {
		return nil, fmt.Errorf("creating repos client: %w", err)
	}

	return &Client{
		packagesClient: pkgClient,
		reposClient:    repoClient,
		userAgent:      cfg.userAgent,
	}, nil
}

// BulkLookup looks up multiple packages by PURL.
// Returns a map keyed by PURL with package data.
// PURLs are processed in batches of 100.
func (c *Client) BulkLookup(ctx context.Context, purls []string) (map[string]*packages.PackageWithRegistry, error) {
	if len(purls) == 0 {
		return map[string]*packages.PackageWithRegistry{}, nil
	}

	results := make(map[string]*packages.PackageWithRegistry)

	for i := 0; i < len(purls); i += MaxBulkLookupSize {
		end := i + MaxBulkLookupSize
		if end > len(purls) {
			end = len(purls)
		}
		batch := purls[i:end]

		resp, err := c.packagesClient.BulkLookupPackagesWithResponse(ctx, packages.BulkLookupPackagesJSONRequestBody{
			Purls: &batch,
		})
		if err != nil {
			return nil, fmt.Errorf("bulk lookup: %w", err)
		}

		if resp.StatusCode() != http.StatusOK {
			if resp.JSON400 != nil && resp.JSON400.Error != nil {
				return nil, fmt.Errorf("bulk lookup failed: %s", *resp.JSON400.Error)
			}
			return nil, fmt.Errorf("bulk lookup failed with status %d", resp.StatusCode())
		}

		if resp.JSON200 != nil {
			for _, pkg := range *resp.JSON200 {
				p := pkg
				results[pkg.Purl] = &p
			}
		}
	}

	return results, nil
}

// Lookup looks up a single package by PURL.
func (c *Client) Lookup(ctx context.Context, purl string) (*packages.PackageWithRegistry, error) {
	results, err := c.BulkLookup(ctx, []string{purl})
	if err != nil {
		return nil, err
	}
	return results[purl], nil
}

// LookupByRegistryAndName looks up a package by registry and name.
func (c *Client) LookupByRegistryAndName(ctx context.Context, registry, name string) (*packages.Package, error) {
	resp, err := c.packagesClient.GetRegistryPackageWithResponse(ctx, registry, name)
	if err != nil {
		return nil, fmt.Errorf("lookup package: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("lookup failed with status %d", resp.StatusCode())
	}

	return resp.JSON200, nil
}

// GetVersion gets a specific version of a package.
func (c *Client) GetVersion(ctx context.Context, registry, name, version string) (*packages.VersionWithDependencies, error) {
	resp, err := c.packagesClient.GetRegistryPackageVersionWithResponse(ctx, registry, name, version)
	if err != nil {
		return nil, fmt.Errorf("get version: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get version failed with status %d", resp.StatusCode())
	}

	return resp.JSON200, nil
}

// GetAllVersions gets all versions of a package.
func (c *Client) GetAllVersions(ctx context.Context, registry, name string) ([]packages.Version, error) {
	var allVersions []packages.Version
	page := 1
	perPage := 100

	for {
		resp, err := c.packagesClient.GetRegistryPackageVersionsWithResponse(ctx, registry, name, &packages.GetRegistryPackageVersionsParams{
			Page:    &page,
			PerPage: &perPage,
		})
		if err != nil {
			return nil, fmt.Errorf("get versions: %w", err)
		}

		if resp.StatusCode() == http.StatusNotFound {
			return nil, nil
		}

		if resp.StatusCode() != http.StatusOK {
			return nil, fmt.Errorf("get versions failed with status %d", resp.StatusCode())
		}

		if resp.JSON200 == nil || len(*resp.JSON200) == 0 {
			break
		}

		allVersions = append(allVersions, *resp.JSON200...)

		if len(*resp.JSON200) < perPage {
			break
		}
		page++
	}

	return allVersions, nil
}

// GetRepository looks up a repository by URL.
func (c *Client) GetRepository(ctx context.Context, url string) (*repos.Repository, error) {
	resp, err := c.reposClient.RepositoriesLookupWithResponse(ctx, &repos.RepositoriesLookupParams{
		Url: &url,
	})
	if err != nil {
		return nil, fmt.Errorf("lookup repository: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("lookup repository failed with status %d", resp.StatusCode())
	}

	return resp.JSON200, nil
}

// ListRegistries returns all available registries.
func (c *Client) ListRegistries(ctx context.Context) ([]packages.Registry, error) {
	resp, err := c.packagesClient.GetRegistriesWithResponse(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("list registries: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("list registries failed with status %d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, nil
	}

	return *resp.JSON200, nil
}
