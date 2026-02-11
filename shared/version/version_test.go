package version

import (
	"testing"
)

func TestParse_ValidPreviewVersion(t *testing.T) {
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
	// Valid format but not in supported list
	_, err := Parse("99.99.future")
	if err == nil {
		t.Error("Expected error for unsupported version")
	}
}

func TestAPIVersion_Before_StableVersions(t *testing.T) {
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
	v := V1_0_Forge_Preview1
	if v.String() != "1.0.forge-preview.1" {
		t.Errorf("Expected %s, got %s", "1.0.forge-preview.1", v.String())
	}
}

func TestSupportedVersionStrings(t *testing.T) {
	strings := SupportedVersionStrings()

	if len(strings) == 0 {
		t.Error("Expected at least one supported version")
	}

	found := false
	for _, s := range strings {
		if s == "1.0.forge-preview.1" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected 1.0.forge-preview.1 in supported versions")
	}
}

func TestIsSupported(t *testing.T) {
	if !IsSupported("1.0.forge-preview.1") {
		t.Error("Expected 1.0.forge-preview.1 to be supported")
	}

	if IsSupported("1.0.forge") {
		t.Error("Expected 1.0.forge to not be supported (not yet stable)")
	}

	if IsSupported("99.99.future") {
		t.Error("Expected 99.99.future to not be supported")
	}
}

func TestMustParse_Valid(t *testing.T) {
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
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParse should panic on invalid input")
		}
	}()

	MustParse("invalid")
}

func TestLatest(t *testing.T) {
	if Latest.Version != "1.0.forge-preview.1" {
		t.Errorf("Expected Latest to be 1.0.forge-preview.1, got %s", Latest.Version)
	}
}

func TestV1_0_Forge_Preview1(t *testing.T) {
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
