package crypto

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestRandAlphanumericBytes_NegativeN(t *testing.T) {
	t.Parallel()
	for _, n := range []int{-1, -32, -1000} {
		b, err := RandAlphanumericBytes(n)
		if err == nil {
			t.Errorf("RandAlphanumericBytes(%d) expected error, got %q", n, b)
		}
		if b != nil {
			t.Errorf("RandAlphanumericBytes(%d) expected nil slice, got %q", n, b)
		}
	}
}

func TestRandAlphanumericString_NegativeN(t *testing.T) {
	t.Parallel()
	s, err := RandAlphanumericString(-1)
	if err == nil {
		t.Fatalf("RandAlphanumericString(-1) expected error, got %q", s)
	}
	if s != "" {
		t.Errorf("RandAlphanumericString(-1) = %q, want empty string", s)
	}
}

// TestRandAlphanumericBytes_LongerThanBuffer covers the refill path: the generator draws 32 bytes at a time and rejects roughly 3% of them, so any n past 32 (API key secrets are 44) makes the loop run more than once.
func TestRandAlphanumericBytes_LongerThanBuffer(t *testing.T) {
	t.Parallel()
	for _, n := range []int{31, 32, 33, 44, 64, 1000} {
		b, err := RandAlphanumericBytes(n)
		if err != nil {
			t.Fatalf("RandAlphanumericBytes(%d) unexpected error: %v", n, err)
		}
		if len(b) != n {
			t.Fatalf("RandAlphanumericBytes(%d) returned %d bytes", n, len(b))
		}
		if !containsOnlyAlphanum(string(b)) {
			t.Fatalf("RandAlphanumericBytes(%d) = %q contains non-alphanumeric bytes", n, b)
		}
	}
}

// TestRejectionBound_ExcludesBiasedBytes pins the rejection constant: charset[b%62] is uniform only while the accepted range is a whole number of charset repetitions, and rejecting more than one full charset's worth would be a needless refill.
func TestRejectionBound_ExcludesBiasedBytes(t *testing.T) {
	t.Parallel()
	if charsetLength != 62 {
		t.Fatalf("charsetLength = %d, want 62", charsetLength)
	}
	if len(charset) != 62 {
		t.Fatalf("len(charset) = %d, want 62", len(charset))
	}
	if maxUnbiased != 248 {
		t.Fatalf("maxUnbiased = %d, want 248", maxUnbiased)
	}
	if int(maxUnbiased)%int(charsetLength) != 0 {
		t.Fatalf("maxUnbiased %d is not a multiple of %d, accepted bytes would be biased", maxUnbiased, charsetLength)
	}
	if 256-int(maxUnbiased) >= int(charsetLength) {
		t.Fatalf("maxUnbiased %d rejects a full charset repetition, it is not the largest usable bound", maxUnbiased)
	}

	// Every accepted byte maps to a charset index, and the counts across a full accepted range are equal.
	counts := make(map[byte]int, 62)
	for b := range 256 {
		if byte(b) >= maxUnbiased {
			continue
		}
		counts[charset[byte(b)%charsetLength]]++
	}
	if len(counts) != 62 {
		t.Fatalf("accepted bytes cover %d charset characters, want 62", len(counts))
	}
	for c, n := range counts {
		if n != int(maxUnbiased)/int(charsetLength) {
			t.Fatalf("character %q is reachable from %d accepted bytes, want %d", string(c), n, int(maxUnbiased)/int(charsetLength))
		}
	}
}

func TestRandHexString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{"length 0", 0, false},
		{"length 1", 1, false},
		{"length 16", 16, false},
		{"length 32", 32, false},
		{"negative length", -1, true},
		{"large negative length", -64, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, err := RandHexString(tt.length)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("RandHexString(%d) expected error, got %q", tt.length, s)
				}
				if s != "" {
					t.Errorf("RandHexString(%d) = %q, want empty string", tt.length, s)
				}
				return
			}
			if err != nil {
				t.Fatalf("RandHexString(%d) unexpected error: %v", tt.length, err)
			}
			if len(s) != tt.length*2 {
				t.Fatalf("RandHexString(%d) = %q (len %d), want len %d", tt.length, s, len(s), tt.length*2)
			}
			if s != strings.ToLower(s) {
				t.Errorf("RandHexString(%d) = %q, want lowercase hex", tt.length, s)
			}
			if _, err := hex.DecodeString(s); err != nil {
				t.Fatalf("RandHexString(%d) = %q is not decodable hex: %v", tt.length, s, err)
			}
		})
	}
}

func TestRandHexString_Uniqueness(t *testing.T) {
	t.Parallel()
	// RandHexString backs the auto-assigned account-user password, so a repeat between two calls would hand two users the same credential.
	seen := make(map[string]bool, 64)
	for range 64 {
		s, err := RandHexString(16)
		if err != nil {
			t.Fatalf("RandHexString() unexpected error: %v", err)
		}
		if seen[s] {
			t.Fatalf("RandHexString() returned a duplicate value: %s", s)
		}
		seen[s] = true
	}
}
