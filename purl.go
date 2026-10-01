package ecosystems

import (
	"context"
	"fmt"
	"strings"

	"github.com/ecosyste-ms/ecosystems-go/packages"
	packageurl "github.com/package-url/packageurl-go"
)

// PURLToRegistry converts a PURL type to the ecosyste.ms registry name.
func PURLToRegistry(purl packageurl.PackageURL) string {
	return purlTypeToRegistry[purl.Type]
}

// PURLToName converts a PURL to the ecosyste.ms package name format.
func PURLToName(purl packageurl.PackageURL) string {
	name := purl.Name

	if purl.Namespace == "" {
		return name
	}

	switch purl.Type {
	case packageurl.TypeMaven:
		// Maven uses colon separator for group:artifact
		return fmt.Sprintf("%s:%s", purl.Namespace, purl.Name)
	case packageurl.TypeApk:
		// APK packages ignore namespace
		return name
	default:
		// Most ecosystems use slash separator
		return fmt.Sprintf("%s/%s", purl.Namespace, purl.Name)
	}
}

// LookupPURL looks up a package by its PURL using the registry/name endpoint.
// This is useful when you need the full Package type rather than PackageWithRegistry.
func (c *Client) LookupPURL(ctx context.Context, purl packageurl.PackageURL) (*packages.Package, error) {
	registry := PURLToRegistry(purl)
	if registry == "" {
		return nil, fmt.Errorf("unsupported PURL type: %s", purl.Type)
	}
	name := PURLToName(purl)
	return c.LookupByRegistryAndName(ctx, registry, name)
}

// GetVersionPURL gets a specific version using a PURL.
func (c *Client) GetVersionPURL(ctx context.Context, purl packageurl.PackageURL) (*packages.VersionWithDependencies, error) {
	if purl.Version == "" {
		return nil, fmt.Errorf("PURL has no version")
	}
	registry := PURLToRegistry(purl)
	if registry == "" {
		return nil, fmt.Errorf("unsupported PURL type: %s", purl.Type)
	}
	name := PURLToName(purl)
	return c.GetVersion(ctx, registry, name, purl.Version)
}

// GetAllVersionsPURL gets all versions for a package using a PURL.
func (c *Client) GetAllVersionsPURL(ctx context.Context, purl packageurl.PackageURL) ([]packages.Version, error) {
	registry := PURLToRegistry(purl)
	if registry == "" {
		return nil, fmt.Errorf("unsupported PURL type: %s", purl.Type)
	}
	name := PURLToName(purl)
	return c.GetAllVersions(ctx, registry, name)
}

// ParsePURL parses a PURL string.
func ParsePURL(s string) (packageurl.PackageURL, error) {
	// Handle bare PURLs without the pkg: scheme
	if !strings.HasPrefix(s, "pkg:") {
		s = "pkg:" + s
	}
	return packageurl.FromString(s)
}

// purlTypeToRegistry maps PURL types to ecosyste.ms registry names.
var purlTypeToRegistry = map[string]string{
	packageurl.TypeAlpm:       "archlinux.org",
	packageurl.TypeApk:        "alpine-edge",
	packageurl.TypeBitbucket:  "",
	packageurl.TypeBitnami:    "",
	packageurl.TypeBower:      "bower.io",
	packageurl.TypeCargo:      "crates.io",
	packageurl.TypeCarthage:   "carthage",
	packageurl.TypeChef:       "supermarket.chef.io",
	packageurl.TypeChocolatey: "chocolatey.org",
	packageurl.TypeClojars:    "clojars.org",
	packageurl.TypeCocoapods:  "cocoapods.org",
	packageurl.TypeComposer:   "packagist.org",
	packageurl.TypeConan:      "conan.io",
	packageurl.TypeConda:      "anaconda.org",
	packageurl.TypeCpan:       "metacpan.org",
	packageurl.TypeCran:       "cran.r-project.org",
	packageurl.TypeDocker:     "hub.docker.com",
	packageurl.TypeElm:        "package.elm-lang.org",
	packageurl.TypeGem:        "rubygems.org",
	packageurl.TypeGeneric:    "",
	packageurl.TypeGithub:     "",
	packageurl.TypeGolang:     "proxy.golang.org",
	packageurl.TypeHackage:    "hackage.haskell.org",
	packageurl.TypeHex:        "hex.pm",
	packageurl.TypeHuggingface: "",
	packageurl.TypeMaven:      "repo1.maven.org",
	packageurl.TypeNPM:        "npmjs.org",
	packageurl.TypeNuget:      "nuget.org",
	packageurl.TypeOCI:        "",
	packageurl.TypePub:        "pub.dev",
	packageurl.TypePyPi:       "pypi.org",
	packageurl.TypeRPM:        "",
	packageurl.TypeSwift:      "swiftpackageindex.com",
	"brew":                    "formulae.brew.sh",
	"deb":                     "debian",
	"julia":                   "juliahub.com",
	"puppet":                  "forge.puppet.com",
}

// SupportedPURLTypes returns all PURL types that have registry mappings.
func SupportedPURLTypes() []string {
	var types []string
	for t, registry := range purlTypeToRegistry {
		if registry != "" {
			types = append(types, t)
		}
	}
	return types
}
