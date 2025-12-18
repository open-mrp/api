package sanitize

import (
	"strings"
)

// Sanitizes a key by showing the first visiblePrefixLength characters and the last visibleSuffixLength characters
// and replacing the rest with *
func SanitizeString(key string, visiblePrefixLength, visibleSuffixLength int) string {
	if visiblePrefixLength < 0 || visibleSuffixLength < 0 {
		return ""
	}

	if key == "" {
		return ""
	}

	if key == "undefined" {
		return "undefined"
	}

	if key == "None" {
		return "None"
	}

	if key == "null" {
		return "null"
	}

	if len(key) < visiblePrefixLength+visibleSuffixLength {
		return key
	}

	prefix := key[:visiblePrefixLength]
	suffix := key[len(key)-visibleSuffixLength:]

	replacementLength := min(len(key)-(visiblePrefixLength+visibleSuffixLength), 4)

	return prefix + strings.Repeat("*", replacementLength) + suffix
}
