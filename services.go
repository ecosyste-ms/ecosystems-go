package ecosystems

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ecosyste-ms/ecosystems-go/advisories"
	"github.com/ecosyste-ms/ecosystems-go/commits"
	"github.com/ecosyste-ms/ecosystems-go/issues"
	"github.com/ecosyste-ms/ecosystems-go/packages"
)

const (
	defaultPerPage       = 100
	defaultMaxPages      = 20
	defaultAcceptContent = "application/json"
)

// LookupPackagesByPURL looks up package records matching a PURL.
func (c *Client) LookupPackagesByPURL(ctx context.Context, purl string) ([]packages.PackageWithRegistry, error) {
	resp, err := c.packagesClient.LookupPackageWithResponse(ctx, &packages.LookupPackageParams{
		Purl: &purl,
	})
	if err != nil {
		return nil, fmt.Errorf("lookup packages by purl: %w", err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("lookup packages by purl failed with status %d", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return nil, nil
	}
	return appendLinkedPages(c, ctx, *resp.JSON200, resp.HTTPResponse, 0)
}

// LookupPackagesByRepositoryURL looks up package records published from a source repository.
func (c *Client) LookupPackagesByRepositoryURL(ctx context.Context, repositoryURL string, maxItems int) ([]packages.PackageWithRegistry, error) {
	resp, err := c.packagesClient.LookupPackageWithResponse(ctx, &packages.LookupPackageParams{
		RepositoryURL: &repositoryURL,
	})
	if err != nil {
		return nil, fmt.Errorf("lookup packages by repository url: %w", err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("lookup packages by repository url failed with status %d", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return nil, nil
	}
	return appendLinkedPages(c, ctx, *resp.JSON200, resp.HTTPResponse, maxItems)
}

// GetAdvisoriesByRepoURL returns advisories associated with a source repository.
func (c *Client) GetAdvisoriesByRepoURL(ctx context.Context, repositoryURL string, maxItems int) ([]advisories.Advisory, error) {
	perPage := perPageForCap(maxItems)
	resp, err := c.advisoriesClient.GetAdvisoriesWithResponse(ctx, &advisories.GetAdvisoriesParams{
		RepositoryURL: &repositoryURL,
		PerPage:       &perPage,
	})
	if err != nil {
		return nil, fmt.Errorf("get advisories by repository url: %w", err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get advisories by repository url failed with status %d", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return nil, nil
	}
	return appendLinkedPages(c, ctx, *resp.JSON200, resp.HTTPResponse, maxItems)
}

// GetDependentPackages returns packages that depend on registry/name.
func (c *Client) GetDependentPackages(ctx context.Context, registry, name string, maxItems int) ([]packages.Package, error) {
	perPage := perPageForCap(maxItems)
	resp, err := c.packagesClient.GetRegistryPackageDependentPackagesWithResponse(
		ctx,
		registry,
		name,
		&packages.GetRegistryPackageDependentPackagesParams{PerPage: &perPage},
	)
	if err != nil {
		return nil, fmt.Errorf("get dependent packages: %w", err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get dependent packages failed with status %d", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return nil, nil
	}
	return appendLinkedPages(c, ctx, *resp.JSON200, resp.HTTPResponse, maxItems)
}

// GetCommitsSummary looks up commit summary metadata for a source repository.
func (c *Client) GetCommitsSummary(ctx context.Context, repositoryURL string) (*commits.Repository, error) {
	resp, err := c.commitsClient.RepositoriesLookupWithResponse(ctx, &commits.RepositoriesLookupParams{
		URL: repositoryURL,
	})
	if err != nil {
		return nil, fmt.Errorf("get commits summary: %w", err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get commits summary failed with status %d", resp.StatusCode())
	}
	return resp.JSON200, nil
}

// GetIssuesSummary looks up issue and pull-request summary metadata for a source repository.
func (c *Client) GetIssuesSummary(ctx context.Context, repositoryURL string) (*issues.Repository, error) {
	resp, err := c.issuesClient.RepositoriesLookupWithResponse(ctx, &issues.RepositoriesLookupParams{
		URL: repositoryURL,
	})
	if err != nil {
		return nil, fmt.Errorf("get issues summary: %w", err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get issues summary failed with status %d", resp.StatusCode())
	}
	return resp.JSON200, nil
}

func perPageForCap(maxItems int) int {
	if maxItems > 0 && maxItems < defaultPerPage {
		return maxItems
	}
	return defaultPerPage
}

func appendLinkedPages[T any](c *Client, ctx context.Context, first []T, resp *http.Response, maxItems int) ([]T, error) {
	out, capped := capItems(first, maxItems)
	if resp == nil || capped {
		return out, nil
	}
	next := nextLink(resp.Header.Get("Link"))
	for page := 1; next != "" && page < defaultMaxPages; page++ {
		var batch []T
		nextResp, err := c.getJSON(ctx, next, &batch)
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
		out, capped = capItems(out, maxItems)
		if capped {
			return out, nil
		}
		next = nextLink(nextResp.Header.Get("Link"))
	}
	if next != "" {
		return out, fmt.Errorf("pagination exceeded max pages %d", defaultMaxPages)
	}
	return out, nil
}

func capItems[T any](items []T, maxItems int) ([]T, bool) {
	if maxItems <= 0 || len(items) <= maxItems {
		return items, false
	}
	return items[:maxItems], true
}

func (c *Client) getJSON(ctx context.Context, rawURL string, out any) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get %s failed with status %d", rawURL, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", defaultAcceptContent)
	if c.fromEmail != "" {
		req.Header.Set("From", c.fromEmail)
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}

// nextLink extracts the URL of the rel="next" entry from an RFC 8288 Link header.
func nextLink(header string) string {
	_, rest, found := strings.Cut(header, "<")
	for found {
		u, after, closed := strings.Cut(rest, ">")
		if !closed {
			return ""
		}
		var params string
		params, rest, found = strings.Cut(after, "<")
		if strings.Contains(params, `rel="next"`) || strings.Contains(params, "rel=next") {
			return u
		}
	}
	return ""
}
