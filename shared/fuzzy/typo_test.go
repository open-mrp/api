package fuzzy

import "testing"

func TestIsTypo(t *testing.T) {
	cases := []struct {
		term, candidate string
		want            bool
	}{
		{"custmer", "customer", true},   // dropped letter
		{"customrs", "customers", true}, // dropped letter
		{"updaet", "update", true},      // adjacent transposition (Levenshtein 2)
		{"custommer", "customer", true}, // doubled letter
		{"create", "create", true},      // identical
		{"create", "delete", false},     // far apart
		{"order", "owner", false},       // 2 edits but a 5-char term only tolerates 1
		{"car", "cart", false},          // too short to fuzzy-match safely
		{"abc", "abcd", false},          // too short
		{"customer", "cust", false},     // length gap wider than tolerance
		{"", "customer", false},
	}
	for _, tc := range cases {
		if got := IsTypo(tc.term, tc.candidate); got != tc.want {
			t.Errorf("IsTypo(%q, %q) = %v, want %v", tc.term, tc.candidate, got, tc.want)
		}
	}
}

func TestAnyTypo(t *testing.T) {
	segments := []string{"create", "customer"}
	if !AnyTypo("custmer", segments) {
		t.Error("custmer should fuzzy-match a candidate")
	}
	if AnyTypo("warehouse", segments) {
		t.Error("warehouse should not fuzzy-match create/customer")
	}
	if AnyTypo("custmer", nil) {
		t.Error("no candidates should never match")
	}
}

// The 2-edit tolerance for 6+ character terms is the loosest branch — these are the words it must still reject.
func TestIsTypo_DistinctLongWords(t *testing.T) {
	t.Parallel()
	cases := []struct {
		term, candidate string
	}{
		{"invoice", "payment"},
		{"orders", "picking"},
		{"shipment", "customer"},
		{"receipt", "picking"},
		{"vendor", "credit"},
		{"customer", "supplier"},
		{"quantity", "warehouse"},
	}
	for _, tc := range cases {
		if IsTypo(tc.term, tc.candidate) {
			t.Errorf("IsTypo(%q, %q) = true, want false: distinct words must not typo-match", tc.term, tc.candidate)
		}
	}
}

// Both the length thresholds and the distance are byte-based, so accented input matches inconsistently:
// dropping an accent costs 2 edits, which the 6-byte "naïve" tolerates and the 5-byte "café" does not.
// Callers that pre-fold to ASCII never see this; the gateway's JSON-field suggestions do not pre-fold.
func TestIsTypo_MultibyteInput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		term, candidate string
		want            bool
	}{
		{"café", "cafe", false},
		{"café", "café", true},
		{"naïve", "naive", true},
		{"日本語", "日本語", true},
	}
	for _, tc := range cases {
		if got := IsTypo(tc.term, tc.candidate); got != tc.want {
			t.Errorf("IsTypo(%q, %q) = %v, want %v", tc.term, tc.candidate, got, tc.want)
		}
	}
}

func TestAnyTypo_MatchesLaterCandidate(t *testing.T) {
	t.Parallel()
	if !AnyTypo("shipmnet", []string{"create", "customer", "shipment"}) {
		t.Error("expected a match against the last candidate")
	}
	if AnyTypo("", []string{"create", "customer"}) {
		t.Error("an empty term should never match")
	}
	if AnyTypo("customer", []string{}) {
		t.Error("an empty candidate list should never match")
	}
}
