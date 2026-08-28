package crypto

import (
	"fmt"
	"hash/crc32"
	"math"
	"strings"
	"testing"
)

// apiKeyChecksumWidth mirrors the width auth-service issues keys with; ParseAPIKey slices exactly this many bytes off the end, so any other length is an unparseable key.
const apiKeyChecksumWidth = 6

func TestCRC32Base62_WidthIsExact(t *testing.T) {
	t.Parallel()
	// A CRC32 sum is at most math.MaxUint32, which needs 6 base-62 digits, so width 6 is both the floor (padding) and the ceiling (no overflow) for every possible input.
	for i := range 5000 {
		in := fmt.Sprintf("key_%d_%s", i, strings.Repeat("x", i%17))
		got := CRC32Base62(in, apiKeyChecksumWidth)
		if len(got) != apiKeyChecksumWidth {
			t.Fatalf("CRC32Base62(%q, %d) = %q (len %d), want len %d", in, apiKeyChecksumWidth, got, len(got), apiKeyChecksumWidth)
		}
	}
}

func TestCRC32Base62_ShortChecksumIsLeftPadded(t *testing.T) {
	t.Parallel()
	// Roughly one input in five hashes below 62^5 and must be padded up to the fixed width; find one so the padding branch is actually exercised.
	const maxFiveDigits = 62 * 62 * 62 * 62 * 62

	var in string
	for i := range 100000 {
		candidate := fmt.Sprintf("pad-%d", i)
		if crc32Castagnoli(candidate) < maxFiveDigits {
			in = candidate
			break
		}
	}
	if in == "" {
		t.Fatal("no input with a sub-62^5 checksum found")
	}

	got := CRC32Base62(in, apiKeyChecksumWidth)
	if len(got) != apiKeyChecksumWidth {
		t.Fatalf("CRC32Base62(%q, %d) = %q, want len %d", in, apiKeyChecksumWidth, got, apiKeyChecksumWidth)
	}
	if got[0] != charset[0] {
		t.Fatalf("CRC32Base62(%q) = %q, want a %q pad character in the leading position", in, got, string(charset[0]))
	}
}

func TestBase62Encode_Boundaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		n     uint64
		width int
		want  string
	}{
		{"zero padded to width", 0, 6, strings.Repeat(string(charset[0]), 6)},
		{"zero at width 1", 0, 1, string(charset[0])},
		{"single digit padded", 61, 6, strings.Repeat(string(charset[0]), 5) + string(charset[61])},
		{"first two-digit value", 62, 6, strings.Repeat(string(charset[0]), 4) + "ba"},
		{"max uint32 fits the width", math.MaxUint32, 6, "eQPpmd"},
		{"width 0 does not pad", 61, 0, string(charset[61])},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := base62Encode(tt.n, tt.width)
			if got != tt.want {
				t.Fatalf("base62Encode(%d, %d) = %q, want %q", tt.n, tt.width, got, tt.want)
			}
			if len(got) < tt.width {
				t.Fatalf("base62Encode(%d, %d) = %q, shorter than the requested width", tt.n, tt.width, got)
			}
		})
	}
}

func TestBase62Encode_RoundTripsThroughDecode(t *testing.T) {
	t.Parallel()
	// The encoding must be reversible for every digit boundary, otherwise two distinct checksums could collapse to one suffix.
	for _, n := range []uint64{0, 1, 61, 62, 63, 3843, 3844, math.MaxUint32 - 1, math.MaxUint32} {
		encoded := base62Encode(n, 6)
		var decoded uint64
		for _, c := range []byte(encoded) {
			idx := strings.IndexByte(charset, c)
			if idx < 0 {
				t.Fatalf("base62Encode(%d) produced non-charset byte %q", n, string(c))
			}
			decoded = decoded*62 + uint64(idx)
		}
		if decoded != n {
			t.Fatalf("base62Encode(%d, 6) = %q decoded back to %d", n, encoded, decoded)
		}
	}
}

func crc32Castagnoli(s string) uint32 {
	return crc32.Checksum([]byte(s), crc32.MakeTable(crc32.Castagnoli))
}
