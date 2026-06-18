package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
)

// HMACSHA256 computes the HMAC-SHA256 of data using the provided key. Returns a 32-byte MAC.
func HMACSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// VerifyHMACSHA256 computes the HMAC-SHA256 of data using the provided key and compares it against expected using a constant-time comparison.
func VerifyHMACSHA256(key, data, expected []byte) bool {
	return hmac.Equal(HMACSHA256(key, data), expected)
}
