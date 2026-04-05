package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// RandAlphanumericBytes returns an output of length n, where each byte is an
// ASCII character chosen uniformly from [a-zA-Z0-9]. The output length n is
// both the number of characters and the number of bytes in the returned slice.
func RandAlphanumericBytes(n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("n must be >= 0")
	}
	out := make([]byte, n)

	// We will allow n=0 and return an empty slice
	if n == 0 {
		return out, nil
	}

	// Generate random bytes in chunks and map them to charset with rejection
	// to ensure uniform selection.
	buf := make([]byte, 32)
	outI := 0

	for outI < n {
		if _, err := rand.Read(buf); err != nil {
			return nil, fmt.Errorf("failed to generate random bytes: %w", err)
		}

		for _, b := range buf {
			// To ensure uniform selection across the charset, we reject any bytes
			// greater than or equal to maxUnbiased. Without this rejection, using
			// modulo would slightly bias the distribution, making some characters
			// more likely than others.
			if b >= maxUnbiased {
				continue // reject to avoid bias
			}

			out[outI] = charset[b%charsetLength]
			outI++
			if outI == n {
				break
			}
		}
	}

	return out, nil
}

// RandAlphanumericString returns a random alphanumeric string of length n,
// implemented on top of RandAlphanumericBytes.
func RandAlphanumericString(n int) (string, error) {
	b, err := RandAlphanumericBytes(n)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// RandHexString returns a hex-encoded string of n random bytes (2*n hex
// characters). Useful for random tokens or passwords.
func RandHexString(n int) (string, error) {
	if n < 0 {
		return "", fmt.Errorf("n must be >= 0")
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}
