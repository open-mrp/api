package sanitize

import (
	"strings"
	"testing"
)

func TestSanitizeString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		key                 string
		visiblePrefixLength int
		visibleSuffixLength int
		expected            string
	}{
		{
			name:                "normal key with prefix and suffix",
			key:                 "abcdefghijklmnop",
			visiblePrefixLength: 3,
			visibleSuffixLength: 2,
			expected:            "abc****op",
		},
		{
			name:                "key shorter than prefix + suffix",
			key:                 "abc",
			visiblePrefixLength: 2,
			visibleSuffixLength: 2,
			expected:            "abc",
		},
		{
			name:                "empty key",
			key:                 "",
			visiblePrefixLength: 3,
			visibleSuffixLength: 2,
			expected:            "",
		},
		{
			name:                "undefined key",
			key:                 "undefined",
			visiblePrefixLength: 3,
			visibleSuffixLength: 2,
			expected:            "undefined",
		},
		{
			name:                "None key",
			key:                 "None",
			visiblePrefixLength: 3,
			visibleSuffixLength: 2,
			expected:            "None",
		},
		{
			name:                "null key",
			key:                 "null",
			visiblePrefixLength: 3,
			visibleSuffixLength: 2,
			expected:            "null",
		},
		{
			name:                "exact prefix + suffix length",
			key:                 "abcdef",
			visiblePrefixLength: 3,
			visibleSuffixLength: 3,
			expected:            "abcdef",
		},
		{
			name:                "negative prefix length",
			key:                 "abcdef",
			visiblePrefixLength: -1,
			visibleSuffixLength: 2,
			expected:            "",
		},
		{
			name:                "negative suffix length",
			key:                 "abcdef",
			visiblePrefixLength: 3,
			visibleSuffixLength: -1,
			expected:            "",
		},
		{
			name:                "both negative lengths",
			key:                 "abcdef",
			visiblePrefixLength: -1,
			visibleSuffixLength: -1,
			expected:            "",
		},
		{
			name:                "replacement limited to 4 characters",
			key:                 "abcdefghijklmnopqrstuvwxyz",
			visiblePrefixLength: 3,
			visibleSuffixLength: 3,
			expected:            "abc****xyz",
		},
		{
			name:                "zero prefix and suffix",
			key:                 "abcdef",
			visiblePrefixLength: 0,
			visibleSuffixLength: 0,
			expected:            "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.key, tt.visiblePrefixLength, tt.visibleSuffixLength)

			if result != tt.expected {
				t.Errorf("SanitizeString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSanitizeString_EdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		key                 string
		visiblePrefixLength int
		visibleSuffixLength int
		validate            func(string) bool
	}{
		{
			name:                "very long key",
			key:                 strings.Repeat("a", 1000),
			visiblePrefixLength: 5,
			visibleSuffixLength: 5,
			validate: func(result string) bool {
				return len(result) == 14 &&
					strings.HasPrefix(result, "aaaaa") &&
					strings.HasSuffix(result, "aaaaa") &&
					strings.Count(result, "*") == 4
			},
		},
		{
			name:                "single character key",
			key:                 "a",
			visiblePrefixLength: 1,
			visibleSuffixLength: 0,
			validate: func(result string) bool {
				return result == "a"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.key, tt.visiblePrefixLength, tt.visibleSuffixLength)

			if !tt.validate(result) {
				t.Errorf("SanitizeString() result '%s' failed validation", result)
			}
		})
	}
}
