package version

import (
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestParse_ValidPreviewVersion(t *testing.T) {
	t.Parallel()
	v, err := Parse("1.0.forge-preview.1")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if v.Version != "1.0.forge-preview.1" {
		t.Errorf("Expected version %s, got %s", "1.0.forge-preview.1", v.Version)
	}

	if v.Minor != 1 {
		t.Errorf("Expected minor %d, got %d", 1, v.Minor)
	}

	if v.Patch != 0 {
		t.Errorf("Expected patch %d, got %d", 0, v.Patch)
	}

	if v.Codename != "forge" {
		t.Errorf("Expected codename %s, got %s", "forge", v.Codename)
	}

	if !v.IsPreview {
		t.Error("Expected IsPreview to be true")
	}

	if v.Preview != 1 {
		t.Errorf("Expected preview %d, got %d", 1, v.Preview)
	}
}

func TestParse_InvalidFormat(t *testing.T) {
	t.Parallel()
	tests := []string{
		"2026-02-01.forge",  // old date-based format
		"1.0",               // missing codename
		"forge",             // missing version numbers
		"1.0.FORGE",         // uppercase codename
		"1.0.forge-preview", // preview without number
		"1.0.123",           // numeric codename
		"v1.0.forge",        // leading v
		"1.0.forge-beta.1",  // non-preview suffix
		"invalid",           // completely invalid
		"",                  // empty
	}

	for _, test := range tests {
		_, err := Parse(test)
		if err == nil {
			t.Errorf("Expected error for invalid version %q", test)
		}
	}
}

func TestParse_UnsupportedVersion(t *testing.T) {
	t.Parallel(
	// Valid format but not in supported list
	)

	_, err := Parse("99.99.future")
	if err == nil {
		t.Error("Expected error for unsupported version")
	}
}

func TestAPIVersion_Before_StableVersions(t *testing.T) {
	t.Parallel()
	v1_0 := APIVersion{
		Version:   "1.0.forge",
		Minor:     1,
		Patch:     0,
		Codename:  "forge",
		Preview:   0,
		IsPreview: false,
	}

	v1_1 := APIVersion{
		Version:   "1.1.forge",
		Minor:     1,
		Patch:     1,
		Codename:  "forge",
		Preview:   0,
		IsPreview: false,
	}

	v2_0 := APIVersion{
		Version:   "2.0.forge",
		Minor:     2,
		Patch:     0,
		Codename:  "forge",
		Preview:   0,
		IsPreview: false,
	}

	if !v1_0.Before(v1_1) {
		t.Error("Expected v1_0.Before(v1_1) to be true")
	}

	if !v1_0.Before(v2_0) {
		t.Error("Expected v1_0.Before(v2_0) to be true")
	}

	if !v1_1.Before(v2_0) {
		t.Error("Expected v1_1.Before(v2_0) to be true")
	}

	if v2_0.Before(v1_0) {
		t.Error("Expected v2_0.Before(v1_0) to be false")
	}

	if v1_0.Before(v1_0) {
		t.Error("Expected v.Before(v) to be false")
	}
}

func TestAPIVersion_Before_PreviewVsStable(t *testing.T) {
	t.Parallel()
	v1_1_preview := APIVersion{
		Version:   "1.1.forge-preview.1",
		Minor:     1,
		Patch:     1,
		Codename:  "forge",
		Preview:   1,
		IsPreview: true,
	}

	v1_1_stable := APIVersion{
		Version:   "1.1.forge",
		Minor:     1,
		Patch:     1,
		Codename:  "forge",
		Preview:   0,
		IsPreview: false,
	}

	// Preview should be before stable for same minor.patch.codename
	if !v1_1_preview.Before(v1_1_stable) {
		t.Error("Expected preview.Before(stable) to be true for same version")
	}

	if v1_1_stable.Before(v1_1_preview) {
		t.Error("Expected stable.Before(preview) to be false for same version")
	}
}

func TestAPIVersion_Before_PreviewVersions(t *testing.T) {
	t.Parallel()
	preview1 := APIVersion{
		Version:   "1.1.forge-preview.1",
		Minor:     1,
		Patch:     1,
		Codename:  "forge",
		Preview:   1,
		IsPreview: true,
	}

	preview2 := APIVersion{
		Version:   "1.1.forge-preview.2",
		Minor:     1,
		Patch:     1,
		Codename:  "forge",
		Preview:   2,
		IsPreview: true,
	}

	if !preview1.Before(preview2) {
		t.Error("Expected preview1.Before(preview2) to be true")
	}

	if preview2.Before(preview1) {
		t.Error("Expected preview2.Before(preview1) to be false")
	}
}

func TestAPIVersion_Before_DifferentCodenames(t *testing.T) {
	t.Parallel()
	alpha := APIVersion{
		Version:   "1.0.alpha",
		Minor:     1,
		Patch:     0,
		Codename:  "alpha",
		Preview:   0,
		IsPreview: false,
	}

	beta := APIVersion{
		Version:   "1.0.beta",
		Minor:     1,
		Patch:     0,
		Codename:  "beta",
		Preview:   0,
		IsPreview: false,
	}

	// Lexicographic comparison: alpha < beta
	if !alpha.Before(beta) {
		t.Error("Expected alpha.Before(beta) to be true")
	}

	if beta.Before(alpha) {
		t.Error("Expected beta.Before(alpha) to be false")
	}
}

func TestAPIVersion_After(t *testing.T) {
	t.Parallel()
	older := APIVersion{
		Version:   "1.0.forge",
		Minor:     1,
		Patch:     0,
		Codename:  "forge",
		Preview:   0,
		IsPreview: false,
	}

	newer := APIVersion{
		Version:   "1.1.forge",
		Minor:     1,
		Patch:     1,
		Codename:  "forge",
		Preview:   0,
		IsPreview: false,
	}

	if !newer.After(older) {
		t.Error("Expected newer.After(older) to be true")
	}

	if older.After(newer) {
		t.Error("Expected older.After(newer) to be false")
	}

	if older.After(older) {
		t.Error("Expected v.After(v) to be false")
	}
}

func TestAPIVersion_Equal(t *testing.T) {
	t.Parallel()
	v1 := V1_0_Forge_Preview1
	v2, _ := Parse("1.0.forge-preview.1")

	if !v1.Equal(v2) {
		t.Error("Expected equal versions to return true")
	}

	other := APIVersion{
		Version:   "1.1.forge",
		Minor:     1,
		Patch:     1,
		Codename:  "forge",
		Preview:   0,
		IsPreview: false,
	}

	if v1.Equal(other) {
		t.Error("Expected different versions to return false")
	}
}

func TestAPIVersion_String(t *testing.T) {
	t.Parallel()
	v := V1_0_Forge_Preview1
	if v.String() != "1.0.forge-preview.1" {
		t.Errorf("Expected %s, got %s", "1.0.forge-preview.1", v.String())
	}
}

func TestSupportedVersionStrings(t *testing.T) {
	t.Parallel()
	strings := SupportedVersionStrings()

	if len(strings) == 0 {
		t.Error("Expected at least one supported version")
	}

	for _, want := range []string{"1.0.forge-preview.1", "1.0.forge-preview.2", "1.0.forge-preview.3"} {
		if !slices.Contains(strings, want) {
			t.Errorf("Expected %s in supported versions", want)
		}
	}
}

func TestIsSupported(t *testing.T) {
	t.Parallel()
	if !IsSupported("1.0.forge-preview.1") {
		t.Error("Expected 1.0.forge-preview.1 to be supported")
	}

	if !IsSupported("1.0.forge-preview.2") {
		t.Error("Expected 1.0.forge-preview.2 to be supported")
	}

	if !IsSupported("1.0.forge-preview.3") {
		t.Error("Expected 1.0.forge-preview.3 to be supported")
	}

	if IsSupported("1.0.forge") {
		t.Error("Expected 1.0.forge to not be supported (not yet stable)")
	}

	if IsSupported("99.99.future") {
		t.Error("Expected 99.99.future to not be supported")
	}
}

func TestMustParse_Valid(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Error("MustParse panicked on valid input")
		}
	}()

	v := MustParse("1.0.forge-preview.1")
	if v.Codename != "forge" {
		t.Error("MustParse returned wrong version")
	}
}

func TestMustParse_Invalid(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParse should panic on invalid input")
		}
	}()

	MustParse("invalid")
}

func TestLatest(t *testing.T) {
	t.Parallel()
	if Latest.Version != "1.0.forge-preview.4" {
		t.Errorf("Expected Latest to be 1.0.forge-preview.4, got %s", Latest.Version)
	}
}

func TestV1_0_Forge_Preview2(t *testing.T) {
	t.Parallel()
	v := V1_0_Forge_Preview2

	if v.Version != "1.0.forge-preview.2" {
		t.Errorf("Expected Version to be 1.0.forge-preview.2, got %s", v.Version)
	}

	if v.Minor != 1 {
		t.Errorf("Expected Minor to be 1, got %d", v.Minor)
	}

	if v.Patch != 0 {
		t.Errorf("Expected Patch to be 0, got %d", v.Patch)
	}

	if v.Codename != "forge" {
		t.Errorf("Expected Codename to be forge, got %s", v.Codename)
	}

	if !v.IsPreview {
		t.Error("Expected IsPreview to be true")
	}

	if v.Preview != 2 {
		t.Errorf("Expected Preview to be 2, got %d", v.Preview)
	}

	if !V1_0_Forge_Preview1.Before(V1_0_Forge_Preview2) {
		t.Error("Expected preview.1 to be before preview.2")
	}
}

func TestV1_0_Forge_Preview1(t *testing.T) {
	t.Parallel()
	v := V1_0_Forge_Preview1

	if v.Version != "1.0.forge-preview.1" {
		t.Errorf("Expected Version to be 1.0.forge-preview.1, got %s", v.Version)
	}

	if v.Minor != 1 {
		t.Errorf("Expected Minor to be 1, got %d", v.Minor)
	}

	if v.Patch != 0 {
		t.Errorf("Expected Patch to be 0, got %d", v.Patch)
	}

	if v.Codename != "forge" {
		t.Errorf("Expected Codename to be forge, got %s", v.Codename)
	}

	if !v.IsPreview {
		t.Error("Expected IsPreview to be true")
	}

	if v.Preview != 1 {
		t.Errorf("Expected Preview to be 1, got %d", v.Preview)
	}
}

func TestParse_BoundaryInvalidFormats(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
	}{
		{"leading whitespace", " 1.0.forge-preview.4"},
		{"trailing whitespace", "1.0.forge-preview.4 "},
		{"trailing newline", "1.0.forge-preview.4\n"},
		{"leading newline", "\n1.0.forge-preview.4"},
		{"internal whitespace", "1.0.forge - preview.4"},
		{"uppercase preview", "1.0.FORGE-PREVIEW.4"},
		{"mixed case codename", "1.0.Forge-preview.4"},
		{"embedded NUL", "1.0.forge-preview.4\x00"},
		{"non-ASCII codename", "1.0.fórge"},
		{"non-ASCII digit", "1.0.forge-preview.٤"},
		{"tab separated", "1.0.forge\t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if err == nil {
				t.Fatalf("expected error for %q", tt.input)
			}
			if !strings.Contains(err.Error(), "invalid version format") {
				t.Errorf("expected an invalid-format error for %q, got %v", tt.input, err)
			}
		})
	}
}

// Well-formed strings that are simply not in Supported must fail differently from malformed ones, so the middleware can tell a client "wrong version" apart from "unparseable header".
func TestParse_WellFormedButUnsupported(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
	}{
		{"leading zero preview number", "1.0.forge-preview.04"},
		{"oversized minor", "99999999999999999999.0.forge"},
		{"oversized preview number", "1.0.forge-preview.99999999999999999999"},
		{"zero preview number", "1.0.forge-preview.0"},
		{"stable of a preview codename", "1.0.forge"},
		{"hyphenated codename", "1.0.forge-crucible"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if err == nil {
				t.Fatalf("expected error for %q", tt.input)
			}
			if !strings.Contains(err.Error(), "unsupported API version") {
				t.Errorf("expected an unsupported-version error for %q, got %v", tt.input, err)
			}
		})
	}
}

// --- version table consistency ---

func TestSupported_ContainsLatest(t *testing.T) {
	t.Parallel()
	if !slices.Contains(Supported, Latest) {
		t.Fatalf("Latest %s is not a member of Supported %v", Latest.Version, SupportedVersionStrings())
	}

	for _, v := range Supported {
		if v.Equal(Latest) {
			continue
		}
		if v.After(Latest) {
			t.Errorf("supported version %s is newer than Latest %s", v.Version, Latest.Version)
		}
	}
}

func TestSupported_StrictlyDescending(t *testing.T) {
	t.Parallel()
	for i := 1; i < len(Supported); i++ {
		newer, older := Supported[i-1], Supported[i]

		if !older.Before(newer) {
			t.Errorf("Supported[%d] (%s) is not before Supported[%d] (%s)", i, older.Version, i-1, newer.Version)
		}
		if newer.Before(older) {
			t.Errorf("Supported[%d] (%s) is before Supported[%d] (%s)", i-1, newer.Version, i, older.Version)
		}
		if older.Equal(newer) {
			t.Errorf("Supported[%d] and Supported[%d] are the same version (%s)", i-1, i, newer.Version)
		}
	}
}

func TestSupported_ParseRoundTrip(t *testing.T) {
	t.Parallel()
	for _, want := range Supported {
		t.Run(want.Version, func(t *testing.T) {
			got, err := Parse(want.Version)
			if err != nil {
				t.Fatalf("Parse(%q) failed: %v", want.Version, err)
			}
			if got != want {
				t.Errorf("Parse(%q) returned %+v, want %+v", want.Version, got, want)
			}
		})
	}
}

// A new version is added by copying an existing literal, which makes it easy to leave Minor/Patch/Preview describing the version that was copied from; Before() then orders the two identically and every downgrade for the release is silently skipped.
func TestSupported_FieldsMatchVersionString(t *testing.T) {
	t.Parallel()
	for _, v := range Supported {
		t.Run(v.Version, func(t *testing.T) {
			var minor, patch, codename, preview string
			isPreview := false

			if matches := previewRegex.FindStringSubmatch(v.Version); matches != nil {
				minor, patch, codename, preview = matches[1], matches[2], matches[3], matches[4]
				isPreview = true
			} else if matches := stableRegex.FindStringSubmatch(v.Version); matches != nil {
				minor, patch, codename, preview = matches[1], matches[2], matches[3], "0"
			} else {
				t.Fatalf("version string %q matches neither the stable nor the preview format", v.Version)
			}

			wantMinor, err := strconv.Atoi(minor)
			if err != nil {
				t.Fatalf("failed to parse minor from %q: %v", v.Version, err)
			}
			wantPatch, err := strconv.Atoi(patch)
			if err != nil {
				t.Fatalf("failed to parse patch from %q: %v", v.Version, err)
			}
			wantPreview, err := strconv.Atoi(preview)
			if err != nil {
				t.Fatalf("failed to parse preview from %q: %v", v.Version, err)
			}

			if v.Minor != wantMinor {
				t.Errorf("Minor is %d, but version string says %d", v.Minor, wantMinor)
			}
			if v.Patch != wantPatch {
				t.Errorf("Patch is %d, but version string says %d", v.Patch, wantPatch)
			}
			if v.Codename != codename {
				t.Errorf("Codename is %q, but version string says %q", v.Codename, codename)
			}
			if v.Preview != wantPreview {
				t.Errorf("Preview is %d, but version string says %d", v.Preview, wantPreview)
			}
			if v.IsPreview != isPreview {
				t.Errorf("IsPreview is %t, but version string says %t", v.IsPreview, isPreview)
			}
		})
	}
}

func TestSupported_NoDuplicateVersionStrings(t *testing.T) {
	t.Parallel()
	seen := make(map[string]struct{}, len(Supported))
	for _, v := range Supported {
		if _, exists := seen[v.Version]; exists {
			t.Errorf("duplicate version string %q in Supported", v.Version)
		}
		seen[v.Version] = struct{}{}
	}
}
