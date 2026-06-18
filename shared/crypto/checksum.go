package crypto

import "hash/crc32"

// CRC32Base62 computes a CRC32-C (Castagnoli) checksum of data and returns it as a base-62 [a-zA-Z0-9] string left-padded to the given width.
func CRC32Base62(data string, width int) string {
	sum := crc32.Checksum([]byte(data), crc32.MakeTable(crc32.Castagnoli))
	return base62Encode(uint64(sum), width)
}

// base62Encode converts n to a fixed-width string using the alphanumeric charset.
func base62Encode(n uint64, width int) string {
	var buf [32]byte
	i := len(buf)

	if n == 0 {
		i--
		buf[i] = charset[0]
	} else {
		for n > 0 {
			i--
			buf[i] = charset[n%62]
			n /= 62
		}
	}

	outLen := len(buf) - i
	if outLen < width {
		pad := make([]byte, width-outLen)
		for j := range pad {
			pad[j] = charset[0]
		}
		return string(pad) + string(buf[i:])
	}

	return string(buf[i:])
}
