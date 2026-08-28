package fuzzy

import (
	"strings"
	"testing"
)

func TestLevenshteinDistance(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		a, b     string
		expected int
	}{
		// Identity
		{"identical strings", "hello", "hello", 0},
		{"both empty", "", "", 0},

		// One empty string
		{"a empty", "", "abc", 3},
		{"b empty", "abc", "", 3},

		// Single operations
		{"single insertion", "cat", "cats", 1},
		{"single deletion", "cats", "cat", 1},
		{"single substitution", "cat", "car", 1},

		// Multiple operations
		{"two substitutions", "abc", "axc", 1},
		{"kitten to sitting", "kitten", "sitting", 3},
		{"saturday to sunday", "saturday", "sunday", 3},
		{"flaw to lawn", "flaw", "lawn", 2},

		// Completely different
		{"no overlap", "abc", "xyz", 3},
		{"different lengths no overlap", "ab", "xyz", 3},

		// Case sensitivity
		{"case differs", "Hello", "hello", 1},
		{"all caps vs lower", "ABC", "abc", 3},

		// Single characters
		{"single same char", "a", "a", 0},
		{"single different char", "a", "b", 1},
		{"single char vs empty", "a", "", 1},

		// Symmetry (distance should be the same regardless of argument order)
		{"symmetry check", "abcdef", "azced", 3},

		// Longer strings
		{"prefix match", "autocompletion", "autocomplete", 3},
		{"transposition", "ab", "ba", 2}, // Levenshtein treats transposition as 2 ops
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LevenshteinDistance(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestLevenshteinDistance_Symmetry(t *testing.T) {
	t.Parallel()
	pairs := [][2]string{
		{"kitten", "sitting"},
		{"saturday", "sunday"},
		{"", "nonempty"},
		{"abc", "xyz"},
		{"flaw", "lawn"},
	}

	for _, pair := range pairs {
		d1 := LevenshteinDistance(pair[0], pair[1])
		d2 := LevenshteinDistance(pair[1], pair[0])
		if d1 != d2 {
			t.Errorf("asymmetric: LevenshteinDistance(%q, %q)=%d but LevenshteinDistance(%q, %q)=%d",
				pair[0], pair[1], d1, pair[1], pair[0], d2)
		}
	}
}

func TestLevenshteinDistance_TriangleInequality(t *testing.T) {
	t.Parallel(
	// For any strings a, b, c: dist(a, c) <= dist(a, b) + dist(b, c)
	)

	triples := [][3]string{
		{"kitten", "sitting", "bitten"},
		{"abc", "axc", "xyz"},
		{"hello", "hallo", "world"},
	}

	for _, triple := range triples {
		ab := LevenshteinDistance(triple[0], triple[1])
		bc := LevenshteinDistance(triple[1], triple[2])
		ac := LevenshteinDistance(triple[0], triple[2])
		if ac > ab+bc {
			t.Errorf("triangle inequality violated: dist(%q,%q)=%d > dist(%q,%q)=%d + dist(%q,%q)=%d",
				triple[0], triple[2], ac, triple[0], triple[1], ab, triple[1], triple[2], bc)
		}
	}
}

func TestFindClosestByLevenshtein(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		target       string
		candidates   []string
		expectedWord string
		expectedDist int
	}{
		{
			name:         "exact match in candidates",
			target:       "hello",
			candidates:   []string{"world", "hello", "help"},
			expectedWord: "hello",
			expectedDist: 0,
		},
		{
			name:         "closest by one edit",
			target:       "cat",
			candidates:   []string{"car", "dog", "fish"},
			expectedWord: "car",
			expectedDist: 1,
		},
		{
			name:         "typo correction",
			target:       "accout",
			candidates:   []string{"account", "action", "address"},
			expectedWord: "account",
			expectedDist: 1,
		},
		{
			name:         "single candidate",
			target:       "test",
			candidates:   []string{"best"},
			expectedWord: "best",
			expectedDist: 1,
		},
		{
			name:         "empty candidates",
			target:       "test",
			candidates:   []string{},
			expectedWord: "",
			expectedDist: 0,
		},
		{
			name:         "nil candidates",
			target:       "test",
			candidates:   nil,
			expectedWord: "",
			expectedDist: 0,
		},
		{
			name:         "picks first among ties",
			target:       "ab",
			candidates:   []string{"ac", "ad"},
			expectedWord: "ac",
			expectedDist: 1,
		},
		{
			name:         "empty target matches shortest candidate",
			target:       "",
			candidates:   []string{"a", "ab", "abc"},
			expectedWord: "a",
			expectedDist: 1,
		},
		{
			name:         "api field suggestion",
			target:       "emial",
			candidates:   []string{"email", "name", "phone", "address"},
			expectedWord: "email",
			expectedDist: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			word, dist := FindClosestByLevenshtein(tt.target, tt.candidates)
			if word != tt.expectedWord || dist != tt.expectedDist {
				t.Errorf("FindClosestByLevenshtein(%q, %v) = (%q, %d), want (%q, %d)",
					tt.target, tt.candidates, word, dist, tt.expectedWord, tt.expectedDist)
			}
		})
	}
}

// The one production caller (api-gateway's unknown-JSON-field suggestion) passes a request-supplied
// target against a fixed set of short field names. There is no length guard, so the full O(m*n)
// matrix runs on whatever arrives; BenchmarkLevenshteinDistance pins what that costs.
func TestFindClosestByLevenshtein_TargetFarLongerThanCandidates(t *testing.T) {
	t.Parallel()
	target := strings.Repeat("a", 4096)
	candidates := []string{"abc", "email", "warehouse"}

	word, dist := FindClosestByLevenshtein(target, candidates)
	if word != "abc" || dist != 4095 {
		t.Errorf("FindClosestByLevenshtein(<4096 a's>, %v) = (%q, %d), want (%q, %d)", candidates, word, dist, "abc", 4095)
	}
	// Distance can never be less than the length difference, so no candidate is a plausible suggestion here.
	if minGap := len(target) - len("warehouse"); dist < minGap {
		t.Errorf("distance %d is below the length-difference lower bound %d", dist, minGap)
	}
}

var lengthTiers = []struct {
	name string
	size int
}{
	{"1KiB", 1 << 10},
	{"64KiB", 1 << 16},
	{"1MiB", 1 << 20},
}

func BenchmarkLevenshteinDistance(b *testing.B) {
	const field = "customerId"
	for _, tier := range lengthTiers {
		b.Run(tier.name, func(b *testing.B) {
			target := strings.Repeat("ab", tier.size/2)
			b.ReportAllocs()
			for b.Loop() {
				LevenshteinDistance(target, field)
			}
		})
	}
}

func BenchmarkFindClosestByLevenshtein(b *testing.B) {
	candidates := []string{"customerId", "email", "name", "phone", "address", "warehouseId"}
	for _, tier := range lengthTiers {
		b.Run(tier.name, func(b *testing.B) {
			target := strings.Repeat("ab", tier.size/2)
			b.ReportAllocs()
			for b.Loop() {
				FindClosestByLevenshtein(target, candidates)
			}
		})
	}
}
