package apikey

import (
	"crypto/rand"
	"hash/crc32"
	"math/big"

	apierror "github.com/augno/api/shared/errors"
)

const keyCharset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
	"0123456789"

// genRandString generates a random key of the given length using the key charset.
func genRandString(length int) (string, error) {
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(keyCharset))))
		if err != nil {
			return "", apierror.NewInternalError(err, "Problem generating API key.")
		}
		result[i] = keyCharset[n.Int64()]
	}
	return string(result), nil
}

// genKeyChecksum generates a checksum for a key using the id and secret.
func genKeyChecksum(id, secret string) string {
	payload := id + "_" + secret
	sum := crc32.Checksum([]byte(payload), crc32.MakeTable(crc32.Castagnoli))
	return genFixedString(uint64(sum), 6)
}

// genFixedString converts a number to a string of the given width.
func genFixedString(n uint64, width int) string {
	var buf [32]byte
	i := len(buf)

	// Write the key charset digits into the buffer from the end (reverse order).
	if n == 0 {
		i--
		buf[i] = keyCharset[0]
	} else {
		for n > 0 {
			i--
			buf[i] = keyCharset[n%62]
			n /= 62
		}
	}

	outLen := len(buf) - i
	if outLen < width {
		// Left-pad with the first charset rune to reach the requested width.
		pad := make([]byte, width-outLen)
		for j := range pad {
			pad[j] = keyCharset[0]
		}
		return string(pad) + string(buf[i:])
	}

	return string(buf[i:])
}
