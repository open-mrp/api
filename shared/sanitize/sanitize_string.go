// Package sanitize provides utilities for masking sensitive string values (API keys, tokens, secrets) in log output, error messages, and user-facing responses. It preserves enough of the original value for humans to identify which key is referenced while hiding the secret material in the middle.
package sanitize

import (
	"strings"
)

// safeVocabulary lists string values that represent "no value" in various languages and serialization formats. When a key matches one of these exactly, it is returned unmasked because it carries no secret — masking it would only confuse the reader.
var safeVocabulary = map[string]struct{}{
	"":          {}, // empty string — nothing to mask
	"undefined": {}, // JavaScript undefined serialized as string
	"None":      {}, // Python None serialized as string
	"null":      {}, // JSON null serialized as string
}

// SanitizeString masks the interior of key, preserving the first visiblePrefixLength characters and the last visibleSuffixLength characters and replacing the hidden portion with asterisks. The number of asterisks is capped at 4 regardless of the actual hidden length, so the output does not leak information about the key's size.
//
// Special cases:
//   - If either length parameter is negative, an empty string is returned as a
//     safety measure (invalid input → no output).
//   - If key matches the [safeVocabulary] (empty string, "undefined", "None",
//     "null"), it is returned unmodified.
//   - If key is shorter than visiblePrefixLength + visibleSuffixLength, it is
//     returned unmodified because there is nothing to mask.
//
// Examples:
//
//	SanitizeString("sk_live_abc123xyz", 7, 3) → "sk_live****xyz"
//	SanitizeString("short", 3, 3)             → "short"       (too short to mask)
//	SanitizeString("null", 3, 2)              → "null"        (safe vocabulary)
//	SanitizeString("abcdef", 0, 0)            → "****"        (fully masked)
func SanitizeString(key string, visiblePrefixLength, visibleSuffixLength int) string {
	if visiblePrefixLength < 0 || visibleSuffixLength < 0 {
		return ""
	}

	// If the key is in the safe vocabulary, return it as is.
	if _, safe := safeVocabulary[key]; safe {
		return key
	}

	if len(key) < visiblePrefixLength+visibleSuffixLength {
		return key
	}

	prefix := key[:visiblePrefixLength]
	suffix := key[len(key)-visibleSuffixLength:]

	replacementLength := min(len(key)-(visiblePrefixLength+visibleSuffixLength), 4)

	return prefix + strings.Repeat("*", replacementLength) + suffix
}
