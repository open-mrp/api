package ratelimit

import (
	"testing"
	"time"
)

func TestLimiter_AllowsUpToCapacityThenBlocks(t *testing.T) {
	base := time.Unix(0, 0)
	l, err := New(&Config{Capacity: 3, RefillPerSec: 1, Now: func() time.Time { return base }}) // freeze time: no refill
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for i := 0; i < 3; i++ {
		if !l.Allow("a") {
			t.Fatalf("call %d should be allowed within capacity", i+1)
		}
	}
	if l.Allow("a") {
		t.Fatal("4th call should be blocked once the bucket is empty")
	}
}

func TestLimiter_RefillsOverTime(t *testing.T) {
	now := time.Unix(0, 0)
	l, err := New(&Config{Capacity: 2, RefillPerSec: 1, Now: func() time.Time { return now }}) // 1 token/sec
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if !l.Allow("a") {
		t.Fatal("first call should be allowed")
	}
	if !l.Allow("a") {
		t.Fatal("second call should be allowed")
	}
	if l.Allow("a") {
		t.Fatal("third call should be blocked")
	}
	now = now.Add(time.Second) // refill one token
	if !l.Allow("a") {
		t.Fatal("after 1s a refilled token should allow one call")
	}
	if l.Allow("a") {
		t.Fatal("only one token refilled; next call blocked")
	}
}

func TestLimiter_KeysAreIndependent(t *testing.T) {
	base := time.Unix(0, 0)
	l, err := New(&Config{Capacity: 1, RefillPerSec: 1, Now: func() time.Time { return base }})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if !l.Allow("a") {
		t.Fatal("first key-a call allowed")
	}
	if l.Allow("a") {
		t.Fatal("second key-a call blocked")
	}
	if !l.Allow("b") {
		t.Fatal("key-b has its own bucket and should be allowed")
	}
}

func TestLimiter_EmptyKeyNeverLimited(t *testing.T) {
	l, err := New(&Config{Capacity: 1, RefillPerSec: 1})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for i := 0; i < 5; i++ {
		if !l.Allow("") {
			t.Fatal("empty key must never be limited")
		}
	}
}

func TestNew_RejectsNonPositiveConfig(t *testing.T) {
	if _, err := New(&Config{Capacity: 0, RefillPerSec: 1}); err == nil {
		t.Fatal("capacity <= 0 must be rejected")
	}
	if _, err := New(&Config{Capacity: 1, RefillPerSec: 0}); err == nil {
		t.Fatal("refill per second <= 0 must be rejected")
	}
}
