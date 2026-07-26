---
title: ecosystems-go
description: Go client library for the ecosyste.ms APIs — packages, repositories, security advisories, commit summaries, and issue summaries across dozens of open-source package registries and forges.
---

# ecosystems-go

This reference is generated with [Sourcey](https://sourcey.com) directly from the library's
Go source on `main`, so every symbol below links back to the exact line it is defined on.

## Installation

```bash
go get github.com/ecosyste-ms/ecosystems-go
```

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ecosyste-ms/ecosystems-go"
)

func main() {
	// A user agent is REQUIRED — NewClient returns an error without one.
	client, err := ecosystems.NewClient("my-app/1.0")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	pkg, err := client.Lookup(ctx, "pkg:gem/rake")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("rake has %d versions\n", pkg.VersionsCount)
}
```

## Conventions worth knowing

These are the non-obvious things a first-time reader of the API reference should carry with them.

- **User agent is mandatory.** `NewClient(userAgent string, ...Option)` errors on an empty user
  agent — the ecosyste.ms APIs identify callers by it.
- **PURLs are the primary key.** Package lookups take a [Package URL](https://github.com/package-url/purl-spec)
  such as `pkg:npm/lodash` or `pkg:gem/rails`. `BulkLookup` resolves many at once; `Lookup`
  resolves one.
- **Optional fields are pointers.** Response structs use `*string` / `*int` for values the API may
  omit (e.g. `*pkg.LatestReleaseNumber`, `*version.Integrity`) — nil-check before dereferencing.
- **Functional options configure the client.** `WithFrom(email)` and `WithAPIKey(key)` identify you
  to raise rate limits; `WithHTTPClient`, `WithBatchSize`, and the per-service `WithPackagesServer` /
  `WithReposServer` / `WithAdvisoriesServer` / `WithCommitsServer` / `WithIssuesServer` override
  endpoints and transport.
- **Retries and rate limits are handled for you.** The client retries idempotent requests with
  backoff and surfaces rate-limit errors with a hint to identify via `WithFrom` or `WithAPIKey`.
- **The API is split by domain package.** `packages`, `repos`, `advisories`, `commits`, and `issues`
  each hold the request/response types (generated from the ecosyste.ms OpenAPI specs); the root
  `ecosystems` package holds the `Client`, its options, and the high-level service methods
  (`LookupPackagesByPURL`, `GetAdvisoriesByRepoURL`, `GetDependentPackages`, `GetCommitsSummary`,
  `GetIssuesSummary`).

## Building this reference

The godoc snapshot is a generated artifact and is not committed — it is regenerated from the
checked-out source every time the docs are built, so the published reference cannot drift from
`main`. From a clean checkout:

```bash
make docs
```

which is equivalent to:

```bash
npx sourcey godoc -m . -o docs/godoc.json   # requires the Go toolchain
cd docs && npx sourcey build                # no Go required; renders docs/dist
```
