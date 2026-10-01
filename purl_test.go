package ecosystems

import (
	"testing"

	packageurl "github.com/package-url/packageurl-go"
)

func TestPURLToRegistry(t *testing.T) {
	tests := []struct {
		purlType string
		expected string
	}{
		{packageurl.TypeNPM, "npmjs.org"},
		{packageurl.TypeGem, "rubygems.org"},
		{packageurl.TypePyPi, "pypi.org"},
		{packageurl.TypeMaven, "repo1.maven.org"},
		{packageurl.TypeCargo, "crates.io"},
		{packageurl.TypeGolang, "proxy.golang.org"},
		{packageurl.TypeNuget, "nuget.org"},
		{packageurl.TypeComposer, "packagist.org"},
		{packageurl.TypeHex, "hex.pm"},
		{packageurl.TypePub, "pub.dev"},
		{packageurl.TypeApk, "alpine-edge"},
		{packageurl.TypeDocker, "hub.docker.com"},
		{"brew", "formulae.brew.sh"},
		{"julia", "juliahub.com"},
	}

	for _, tt := range tests {
		t.Run(tt.purlType, func(t *testing.T) {
			purl := packageurl.PackageURL{Type: tt.purlType}
			got := PURLToRegistry(purl)
			if got != tt.expected {
				t.Errorf("PURLToRegistry(%s) = %q, want %q", tt.purlType, got, tt.expected)
			}
		})
	}
}

func TestPURLToName(t *testing.T) {
	tests := []struct {
		name      string
		purl      packageurl.PackageURL
		expected  string
	}{
		{
			name:     "simple npm package",
			purl:     packageurl.PackageURL{Type: packageurl.TypeNPM, Name: "lodash"},
			expected: "lodash",
		},
		{
			name:     "scoped npm package",
			purl:     packageurl.PackageURL{Type: packageurl.TypeNPM, Namespace: "babel", Name: "core"},
			expected: "babel/core",
		},
		{
			name:     "maven package",
			purl:     packageurl.PackageURL{Type: packageurl.TypeMaven, Namespace: "org.apache.commons", Name: "commons-lang3"},
			expected: "org.apache.commons:commons-lang3",
		},
		{
			name:     "gem package",
			purl:     packageurl.PackageURL{Type: packageurl.TypeGem, Name: "rails"},
			expected: "rails",
		},
		{
			name:     "pypi package",
			purl:     packageurl.PackageURL{Type: packageurl.TypePyPi, Name: "requests"},
			expected: "requests",
		},
		{
			name:     "go package with namespace",
			purl:     packageurl.PackageURL{Type: packageurl.TypeGolang, Namespace: "github.com/go-git", Name: "go-git"},
			expected: "github.com/go-git/go-git",
		},
		{
			name:     "apk ignores namespace",
			purl:     packageurl.PackageURL{Type: packageurl.TypeApk, Namespace: "alpine", Name: "curl"},
			expected: "curl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PURLToName(tt.purl)
			if got != tt.expected {
				t.Errorf("PURLToName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParsePURL(t *testing.T) {
	tests := []struct {
		input    string
		wantType string
		wantName string
		wantVer  string
		wantErr  bool
	}{
		{
			input:    "pkg:gem/rails@7.0.0",
			wantType: "gem",
			wantName: "rails",
			wantVer:  "7.0.0",
		},
		{
			input:    "gem/rails@7.0.0",
			wantType: "gem",
			wantName: "rails",
			wantVer:  "7.0.0",
		},
		{
			input:    "pkg:npm/@babel/core@7.20.0",
			wantType: "npm",
			wantName: "core",
			wantVer:  "7.20.0",
		},
		{
			input:    "pkg:maven/org.apache.commons/commons-lang3@3.12.0",
			wantType: "maven",
			wantName: "commons-lang3",
			wantVer:  "3.12.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParsePURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}
			if got.Version != tt.wantVer {
				t.Errorf("Version = %q, want %q", got.Version, tt.wantVer)
			}
		})
	}
}

func TestSupportedPURLTypes(t *testing.T) {
	types := SupportedPURLTypes()
	if len(types) == 0 {
		t.Error("SupportedPURLTypes() returned empty slice")
	}

	// Check that common types are included
	typeSet := make(map[string]bool)
	for _, typ := range types {
		typeSet[typ] = true
	}

	required := []string{"npm", "gem", "pypi", "maven", "cargo", "golang"}
	for _, r := range required {
		if !typeSet[r] {
			t.Errorf("SupportedPURLTypes() missing %q", r)
		}
	}
}
