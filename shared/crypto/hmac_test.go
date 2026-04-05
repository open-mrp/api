package crypto

import (
	"testing"
)

func TestHMACSHA256_RoundTrip(t *testing.T) {
	t.Parallel()
	key := []byte("test-secret-key")
	data := []byte("hello world")

	mac := HMACSHA256(key, data)

	if !VerifyHMACSHA256(key, data, mac) {
		t.Error("VerifyHMACSHA256 should return true for matching MAC")
	}
}

func TestHMACSHA256_OutputLength(t *testing.T) {
	t.Parallel()
	key := []byte("key")
	data := []byte("data")

	mac := HMACSHA256(key, data)

	if len(mac) != 32 {
		t.Errorf("HMACSHA256 should return 32 bytes, got %d", len(mac))
	}
}

func TestHMACSHA256_Deterministic(t *testing.T) {
	t.Parallel()
	key := []byte("key")
	data := []byte("data")

	mac1 := HMACSHA256(key, data)
	mac2 := HMACSHA256(key, data)

	if string(mac1) != string(mac2) {
		t.Error("HMACSHA256 should be deterministic")
	}
}

func TestVerifyHMACSHA256_WrongData(t *testing.T) {
	t.Parallel()
	key := []byte("key")
	mac := HMACSHA256(key, []byte("correct data"))

	if VerifyHMACSHA256(key, []byte("wrong data"), mac) {
		t.Error("VerifyHMACSHA256 should return false for wrong data")
	}
}

func TestVerifyHMACSHA256_WrongKey(t *testing.T) {
	t.Parallel()
	data := []byte("data")
	mac := HMACSHA256([]byte("correct-key"), data)

	if VerifyHMACSHA256([]byte("wrong-key"), data, mac) {
		t.Error("VerifyHMACSHA256 should return false for wrong key")
	}
}
