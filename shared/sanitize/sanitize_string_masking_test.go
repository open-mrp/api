package sanitize

import (
	"strings"
	"testing"
)

func TestSanitizeString_InteriorNeverSurvives(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		key                 string
		visiblePrefixLength int
		visibleSuffixLength int
	}{
		{"live secret key", "sk_live_51H8fQ2LkD9mNpQrS", 7, 3},
		{"api key with mode segment", "mrp_sk_production_abcdefghijklmnop", 20, 4},
		{"one character to hide", "abcdefg", 3, 3},
		{"no visible characters", "topsecretvalue", 0, 0},
		{"prefix only", "topsecretvalue", 4, 0},
		{"suffix only", "topsecretvalue", 0, 4},
		{"multi-byte value", strings.Repeat("é", 40), 4, 4},
		{"very long value", strings.Repeat("z", 4096), 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := SanitizeString(tt.key, tt.visiblePrefixLength, tt.visibleSuffixLength)

			if got == tt.key {
				t.Fatalf("SanitizeString() returned the input unmasked: %q", got)
			}
			interior := tt.key[tt.visiblePrefixLength : len(tt.key)-tt.visibleSuffixLength]
			if strings.Contains(got, interior) {
				t.Fatalf("SanitizeString() = %q still contains the hidden interior %q", got, interior)
			}
			if maxLen := tt.visiblePrefixLength + tt.visibleSuffixLength + 4; len(got) > maxLen {
				t.Fatalf("SanitizeString() = %q is %d bytes, longer than the %d byte masked form", got, len(got), maxLen)
			}
		})
	}
}

func TestSanitizeString_OutputDoesNotRevealLength(t *testing.T) {
	t.Parallel()
	// Two secrets sharing a prefix and suffix must mask to the same string; a variable run of asterisks would publish the hidden length.
	short := SanitizeString("sk_live_"+strings.Repeat("a", 8)+"tail", 8, 4)
	long := SanitizeString("sk_live_"+strings.Repeat("a", 8000)+"tail", 8, 4)

	if short != long {
		t.Fatalf("masked forms differ by input length: %q vs %q", short, long)
	}
	if strings.Count(short, "*") != 4 {
		t.Fatalf("SanitizeString() = %q, want exactly 4 asterisks", short)
	}
}

func TestSanitizeString_SafeVocabularyIsExhaustive(t *testing.T) {
	t.Parallel()
	// Only the placeholder values pass through untouched; anything else that merely resembles them must still be masked.
	for _, safe := range []string{"", "undefined", "None", "null"} {
		if got := SanitizeString(safe, 2, 2); got != safe {
			t.Errorf("SanitizeString(%q) = %q, want it returned unchanged", safe, got)
		}
	}

	for _, unsafe := range []string{"nulled", "Undefined", "null ", "none-of-it", "undefined_key"} {
		if got := SanitizeString(unsafe, 2, 2); got == unsafe {
			t.Errorf("SanitizeString(%q) = %q, want it masked; only the exact placeholders are safe", unsafe, got)
		}
	}
}
