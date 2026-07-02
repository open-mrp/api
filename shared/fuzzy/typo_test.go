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
