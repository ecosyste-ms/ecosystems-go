// Package ecosystems provides a client for the ecosyste.ms APIs.
//
// This package wraps the generated OpenAPI clients for packages.ecosyste.ms
// and repos.ecosyste.ms, providing a higher-level API for common operations.
package ecosystems

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ecosyste-ms/ecosystems-go/advisories"
	"github.com/ecosyste-ms/ecosystems-go/commits"
	"github.com/ecosyste-ms/ecosystems-go/issues"
	"github.com/ecosyste-ms/ecosystems-go/packages"
	"github.com/ecosyste-ms/ecosystems-go/repos"
)

// streamErrHint is appended when the underlying transport reports a
// stream-level rejection (HTTP/2 INTERNAL_ERROR, GOAWAY, etc). These are
// not rate limits — the server is closing the connection without a status
// code, often because the request is too large or the shared pool is
// overloaded.
const streamErrHint = " (the request was rejected before a response; " +
	"try identifying with WithFrom(email) or WithAPIKey, or a smaller WithBatchSize)"

// rateLimitHint is appended to HTTP 429 responses, which ecosyste.ms
// returns specifically for rate-limited requests.
const rateLimitHint = " (rate limited; try identifying with WithFrom(email) or WithAPIKey)"

// looksLikeStreamErr returns true when the transport-level error suggests
// the server closed the HTTP/2 stream without sending a proper response.
func looksLikeStreamErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "stream error"):
		return true
	case strings.Contains(msg, "INTERNAL_ERROR"):
		return true
	case strings.Contains(msg, "GOAWAY"):
		return true
	case strings.Contains(msg, "ENHANCE_YOUR_CALM"):
		return true
	}
	return false
}

// errBulkLookupStream wraps a transport-level stream error with a hint,
// while preserving the original via errors.Is/As.
func errBulkLookupStream(err error) error {
	return fmt.Errorf("bulk lookup: %w"+streamErrHint, err)
}

// errBulkLookupStatus formats a non-200 response, appending a hint when
// the status code is one we can give actionable guidance for.
func errBulkLookupStatus(code int, detail string) error {
	base := fmt.Sprintf("bulk lookup failed with status %d", code)
	if detail != "" {
		base = fmt.Sprintf("bulk lookup failed: %s", detail)
	}
	if code == http.StatusTooManyRequests {
		return errors.New(base + rateLimitHint)
	}
	return errors.New(base)
}

const (
	DefaultPackagesServer   = "https://packages.ecosyste.ms/api/v1"
	DefaultReposServer      = "https://repos.ecosyste.ms/api/v1"
	DefaultAdvisoriesServer = "https://advisories.ecosyste.ms/api/v1"
	DefaultCommitsServer    = "https://commits.ecosyste.ms/api/v1"
	DefaultIssuesServer     = "https://issues.ecosyste.ms/api/v1"
	DefaultTimeout          = 30 * time.Second
	MaxBulkLookupSize       = 100

	defaultRetryAttempts  = 3
	defaultRetryBaseDelay = 200 * time.Millisecond
	defaultRetryMaxDelay  = 2 * time.Second
	backoffFactor         = 2
	jitterDivisor         = 4
)

type Client struct {
	packagesClient   *packages.ClientWithResponses
	reposClient      *repos.ClientWithResponses
	advisoriesClient *advisories.ClientWithResponses
	commitsClient    *commits.ClientWithResponses
	issuesClient     *issues.ClientWithResponses
	httpClient       *http.Client
	userAgent        string
	fromEmail        string
	apiKey           string
	batchSize        int
}

type Option func(*clientConfig)

type clientConfig struct {
	packagesServer   string
	reposServer      string
	advisoriesServer string
	commitsServer    string
	issuesServer     string
	httpClient       *http.Client
	userAgent        string
	fromEmail        string
	apiKey           string
	batchSize        int
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

func WithAdvisoriesServer(server string) Option {
	return func(c *clientConfig) {
		c.advisoriesServer = server
	}
}

func WithCommitsServer(server string) Option {
	return func(c *clientConfig) {
		c.commitsServer = server
	}
}

func WithIssuesServer(server string) Option {
	return func(c *clientConfig) {
		c.issuesServer = server
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

// WithBatchSize sets the number of PURLs sent per BulkLookup request.
// Values <= 0 or greater than MaxBulkLookupSize fall back to MaxBulkLookupSize.
// Useful for clients that need to stay within stricter server-side limits.
func WithBatchSize(size int) Option {
	return func(c *clientConfig) {
		c.batchSize = size
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
		Transport: &retryRoundTripper{
			base:      transport,
			attempts:  defaultRetryAttempts,
			baseDelay: defaultRetryBaseDelay,
			maxDelay:  defaultRetryMaxDelay,
		},
		Timeout: DefaultTimeout,
	}
}

type retryRoundTripper struct {
	base      http.RoundTripper
	attempts  int
	baseDelay time.Duration
	maxDelay  time.Duration
}

func (rt *retryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := rt.base
	if base == nil {
		base = http.DefaultTransport
	}
	if !retryableMethod(req.Method) {
		return base.RoundTrip(req)
	}

	attempts := rt.attempts
	if attempts <= 0 {
		attempts = defaultRetryAttempts
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := req.Context().Err(); err != nil {
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, err
		}
		resp, err := base.RoundTrip(req.Clone(req.Context()))
		if err == nil && !retryableStatus(resp.StatusCode) {
			return resp, nil
		}
		if attempt == attempts {
			if err != nil {
				return nil, err
			}
			return resp, nil
		}

		delay := backoffDelay(attempt, rt.baseDelay, rt.maxDelay)
		if err != nil {
			lastErr = err
		} else {
			delay = retryAfterDelay(resp.Header.Get("Retry-After"), delay, rt.maxDelay)
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
		if err := sleepContext(req.Context(), delay); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func retryableMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}

func retryableStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func retryAfterDelay(header string, fallback, maxDelay time.Duration) time.Duration {
	if header == "" {
		return fallback
	}
	if seconds, err := strconv.Atoi(header); err == nil {
		return capDelay(time.Duration(seconds)*time.Second, maxDelay)
	}
	if at, err := http.ParseTime(header); err == nil {
		delay := time.Until(at)
		if delay > 0 {
			return capDelay(delay, maxDelay)
		}
		return 0
	}
	return fallback
}

func backoffDelay(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	if baseDelay <= 0 {
		baseDelay = defaultRetryBaseDelay
	}
	if maxDelay <= 0 {
		maxDelay = defaultRetryMaxDelay
	}
	delay := baseDelay
	for range attempt - 1 {
		delay *= backoffFactor
		if delay >= maxDelay {
			return maxDelay
		}
	}
	return capDelay(jitter(delay), maxDelay)
}

func capDelay(delay, maxDelay time.Duration) time.Duration {
	if maxDelay > 0 && delay > maxDelay {
		return maxDelay
	}
	return delay
}

func jitter(delay time.Duration) time.Duration {
	if delay <= 0 {
		return 0
	}
	spread := delay / jitterDivisor
	if spread <= 0 {
		return delay
	}
	return delay + time.Duration(rand.Int64N(int64(spread)))
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// NewClient creates a new ecosyste.ms API client.
// The userAgent parameter is required and should identify your application.
func NewClient(userAgent string, opts ...Option) (*Client, error) {
	if userAgent == "" {
		return nil, fmt.Errorf("userAgent is required")
	}

	cfg := &clientConfig{
		packagesServer:   DefaultPackagesServer,
		reposServer:      DefaultReposServer,
		advisoriesServer: DefaultAdvisoriesServer,
		commitsServer:    DefaultCommitsServer,
		issuesServer:     DefaultIssuesServer,
		httpClient:       defaultHTTPClient(),
		userAgent:        userAgent,
		batchSize:        MaxBulkLookupSize,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.batchSize <= 0 || cfg.batchSize > MaxBulkLookupSize {
		cfg.batchSize = MaxBulkLookupSize
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

	advisoriesClient, err := advisories.NewClientWithResponses(
		cfg.advisoriesServer,
		advisories.WithHTTPClient(cfg.httpClient),
		advisories.WithRequestEditorFn(addHeaders),
	)
	if err != nil {
		return nil, fmt.Errorf("creating advisories client: %w", err)
	}

	commitsClient, err := commits.NewClientWithResponses(
		cfg.commitsServer,
		commits.WithHTTPClient(cfg.httpClient),
		commits.WithRequestEditorFn(addHeaders),
	)
	if err != nil {
		return nil, fmt.Errorf("creating commits client: %w", err)
	}

	issuesClient, err := issues.NewClientWithResponses(
		cfg.issuesServer,
		issues.WithHTTPClient(cfg.httpClient),
		issues.WithRequestEditorFn(addHeaders),
	)
	if err != nil {
		return nil, fmt.Errorf("creating issues client: %w", err)
	}

	return &Client{
		packagesClient:   pkgClient,
		reposClient:      repoClient,
		advisoriesClient: advisoriesClient,
		commitsClient:    commitsClient,
		issuesClient:     issuesClient,
		httpClient:       cfg.httpClient,
		userAgent:        cfg.userAgent,
		fromEmail:        cfg.fromEmail,
		apiKey:           cfg.apiKey,
		batchSize:        cfg.batchSize,
	}, nil
}

// BulkLookup looks up multiple packages by PURL.
// Returns a map keyed by PURL with package data.
// PURLs are processed in batches of the configured size (defaults to MaxBulkLookupSize).
// Use WithBatchSize to lower the batch size when the server rejects larger requests.
func (c *Client) BulkLookup(ctx context.Context, purls []string) (map[string]*packages.PackageWithRegistry, error) {
	if len(purls) == 0 {
		return map[string]*packages.PackageWithRegistry{}, nil
	}

	batchSize := c.batchSize
	if batchSize <= 0 || batchSize > MaxBulkLookupSize {
		batchSize = MaxBulkLookupSize
	}

	results := make(map[string]*packages.PackageWithRegistry)

	for i := 0; i < len(purls); i += batchSize {
		end := i + batchSize
		if end > len(purls) {
			end = len(purls)
		}
		batch := purls[i:end]

		resp, err := c.packagesClient.BulkLookupPackagesWithResponse(ctx, packages.BulkLookupPackagesJSONRequestBody{
			PURLs: batch,
		})
		if err != nil {
			if looksLikeStreamErr(err) {
				return nil, errBulkLookupStream(err)
			}
			return nil, fmt.Errorf("bulk lookup: %w", err)
		}

		if resp.StatusCode() != http.StatusOK {
			detail := ""
			if resp.JSON400 != nil && resp.JSON400.Error != nil {
				detail = *resp.JSON400.Error
			}
			return nil, errBulkLookupStatus(resp.StatusCode(), detail)
		}

		if resp.JSON200 != nil {
			for _, pkg := range *resp.JSON200 {
				p := pkg
				results[pkg.PURL] = &p
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
		URL: &url,
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
