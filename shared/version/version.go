package version

import (
	"fmt"
	"regexp"
	"strings"
)

// APIVersion represents an API version with minor, patch, codename, and optional preview.
// Stable format: <minor>.<patch>.<codename> (e.g., 1.0.forge)
// Preview format: <minor>.<patch>.<codename>-preview.<n> (e.g., 1.1.forge-preview.1)
type APIVersion struct {
	Version   string // Full version string "1.0.forge" or "1.1.forge-preview.1"
	Minor     int    // Minor version number
	Patch     int    // Patch version number
	Codename  string // Version codename (e.g., "forge")
	Preview   int    // Preview number (0 for stable releases)
	IsPreview bool   // True if this is a preview version
}

// Defined API versions (newest to oldest)
var (
	V1_0_Forge_Preview4 = APIVersion{
		Version:   "1.0.forge-preview.4",
		Minor:     1,
		Patch:     0,
		Codename:  "forge",
		Preview:   4,
		IsPreview: true,
	}

	V1_0_Forge_Preview3 = APIVersion{
		Version:   "1.0.forge-preview.3",
		Minor:     1,
		Patch:     0,
		Codename:  "forge",
		Preview:   3,
		IsPreview: true,
	}

	V1_0_Forge_Preview2 = APIVersion{
		Version:   "1.0.forge-preview.2",
		Minor:     1,
		Patch:     0,
		Codename:  "forge",
		Preview:   2,
		IsPreview: true,
	}

	V1_0_Forge_Preview1 = APIVersion{
		Version:   "1.0.forge-preview.1",
		Minor:     1,
		Patch:     0,
		Codename:  "forge",
		Preview:   1,
		IsPreview: true,
	}

	// Latest is the current/default API version
	Latest = V1_0_Forge_Preview4

	// Supported lists all supported API versions
	Supported = []APIVersion{V1_0_Forge_Preview4, V1_0_Forge_Preview3, V1_0_Forge_Preview2, V1_0_Forge_Preview1}
)

// Regex patterns for version formats
var (
	// stableRegex matches <minor>.<patch>.<codename> format (e.g., 1.0.forge)
	stableRegex = regexp.MustCompile(`^([0-9]+)\.([0-9]+)\.([a-z][a-z0-9-]*)$`)
	// previewRegex matches <minor>.<patch>.<codename>-preview.<n> format (e.g., 1.1.forge-preview.1)
	previewRegex = regexp.MustCompile(`^([0-9]+)\.([0-9]+)\.([a-z][a-z0-9-]*)-preview\.([0-9]+)$`)
)

// Parse parses a version string into an APIVersion.
// Returns an error if the format is invalid or the version is not supported.
func Parse(s string) (APIVersion, error) {
	// Try preview format first
	if matches := previewRegex.FindStringSubmatch(s); matches != nil {
		// Check if this is a supported version
		for _, v := range Supported {
			if v.Version == s {
				return v, nil
			}
		}

		return APIVersion{}, fmt.Errorf("unsupported API version: %s", s)
	}

	// Try stable format
	if matches := stableRegex.FindStringSubmatch(s); matches != nil {
		// Check if this is a supported version
		for _, v := range Supported {
			if v.Version == s {
				return v, nil
			}
		}

		return APIVersion{}, fmt.Errorf("unsupported API version: %s", s)
	}

	return APIVersion{}, fmt.Errorf("invalid version format: %s (expected <minor>.<patch>.<codename> or <minor>.<patch>.<codename>-preview.<n>)", s)
}

// MustParse parses a version string and panics if it fails.
// Use only for known-valid version strings.
func MustParse(s string) APIVersion {
	v, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

// Before returns true if v is before other (older version).
// Comparison order: minor → patch → codename → preview (preview < stable)
func (v APIVersion) Before(other APIVersion) bool {
	// Compare minor version
	if v.Minor != other.Minor {
		return v.Minor < other.Minor
	}

	// Compare patch version
	if v.Patch != other.Patch {
		return v.Patch < other.Patch
	}

	// Compare codename lexicographically
	cmp := strings.Compare(v.Codename, other.Codename)
	if cmp != 0 {
		return cmp < 0
	}

	// Same minor.patch.codename: preview < stable
	// If both are previews, compare preview numbers
	if v.IsPreview && other.IsPreview {
		return v.Preview < other.Preview
	}

	// Preview is before stable
	if v.IsPreview && !other.IsPreview {
		return true
	}

	// Stable is not before preview
	return false
}

// After returns true if v is after other (newer version).
func (v APIVersion) After(other APIVersion) bool {
	return other.Before(v)
}

// Equal returns true if v and other are the same version.
func (v APIVersion) Equal(other APIVersion) bool {
	return v.Version == other.Version
}

// String returns the full version string.
func (v APIVersion) String() string {
	return v.Version
}

// SupportedVersionStrings returns a slice of all supported version strings.
func SupportedVersionStrings() []string {
	result := make([]string, len(Supported))
	for i, v := range Supported {
		result[i] = v.Version
	}
	return result
}

// IsSupported returns true if the given version string is supported.
func IsSupported(s string) bool {
	for _, v := range Supported {
		if v.Version == s {
			return true
		}
	}
	return false
}
