# ecosystems-go

Go client library for the [ecosyste.ms](https://ecosyste.ms) APIs. See [API documentation](https://ecosyste.ms/api) for details.

## Installation

```bash
go get github.com/ecosyste-ms/ecosystems-go
```

## Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/ecosyste-ms/ecosystems-go"
)

func main() {
    // User agent is required - identify your application
    client, err := ecosystems.NewClient("my-app/1.0")
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Bulk lookup packages by PURL
    results, err := client.BulkLookup(ctx, []string{
        "pkg:gem/rails",
        "pkg:npm/lodash",
        "pkg:pypi/requests",
    })
    if err != nil {
        log.Fatal(err)
    }

    for purl, pkg := range results {
        fmt.Printf("%s: %s (%s)\n", purl, pkg.Name, *pkg.LatestReleaseNumber)
    }

    // Lookup a single package
    pkg, err := client.Lookup(ctx, "pkg:gem/rake")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("rake has %d versions\n", pkg.VersionsCount)

    // Get a specific version
    version, err := client.GetVersion(ctx, "rubygems.org", "rake", "13.0.0")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("rake 13.0.0 integrity: %s\n", *version.Integrity)

    // Get all versions
    versions, err := client.GetAllVersions(ctx, "rubygems.org", "rake")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("rake has %d versions\n", len(versions))
}
```

## PURL Helpers

The library includes helpers for working with Package URLs:

```go
import (
    "github.com/ecosyste-ms/ecosystems-go"
    packageurl "github.com/package-url/packageurl-go"
)

// Parse a PURL string (handles with or without pkg: prefix)
purl, err := ecosystems.ParsePURL("gem/rails@7.0.0")

// Convert PURL to ecosyste.ms registry name
registry := ecosystems.PURLToRegistry(purl) // "rubygems.org"

// Convert PURL to ecosyste.ms package name format
name := ecosystems.PURLToName(purl) // "rails"

// Lookup using PURL directly
pkg, err := client.LookupPURL(ctx, purl)
version, err := client.GetVersionPURL(ctx, purl)
versions, err := client.GetAllVersionsPURL(ctx, purl)
```

## Options

```go
client, err := ecosystems.NewClient("my-app/1.0",
    ecosystems.WithFrom("you@example.com"),      // From header (email)
    ecosystems.WithAPIKey("your-api-key"),       // API key for higher rate limits
    ecosystems.WithHTTPClient(customHTTPClient),
    ecosystems.WithPackagesServer("https://custom.packages.server"),
    ecosystems.WithReposServer("https://custom.repos.server"),
)
```

## Generated Code

The `packages/` and `repos/` directories contain generated OpenAPI clients. To regenerate after spec updates:

```bash
make update-specs  # Download latest OpenAPI specs
make generate      # Regenerate Go clients
```

## Testing

```bash
make test              # Unit tests
make test-integration  # Integration tests (hits live API)
```

## License

MIT
