package ptrutil

import "testing"

// --- ValOrDefault ---

func TestValOrDefault_NonNilPointer(t *testing.T) {
	v := new("actual")
	got := ValOrDefault(v, "fallback")
	if got != "actual" {
		t.Errorf("expected %q, got %q", "actual", got)
	}
}

func TestValOrDefault_NilPointer(t *testing.T) {
	got := ValOrDefault(nil, "fallback")
	if got != "fallback" {
		t.Errorf("expected %q, got %q", "fallback", got)
	}
}

func TestValOrDefault_NilPointerInt(t *testing.T) {
	got := ValOrDefault(nil, 99)
	if got != 99 {
		t.Errorf("expected 99, got %d", got)
	}
}

func TestValOrDefault_ZeroValuePointer(t *testing.T) {
	p := new(0)
	got := ValOrDefault(p, 99)
	if got != 0 {
		t.Errorf("expected 0 (the pointed-to value), got %d", got)
	}
}

func TestValOrDefault_EmptyStringPointer(t *testing.T) {
	p := new("")
	got := ValOrDefault(p, "default")
	if got != "" {
		t.Errorf("expected empty string (the pointed-to value), got %q", got)
	}
}

func TestValOrDefault_FalsePointer(t *testing.T) {
	p := new(false)
	got := ValOrDefault(p, true)
	if got != false {
		t.Errorf("expected false (the pointed-to value), got %v", got)
	}
}

// --- ValOrDefaultFunc ---

func TestValOrDefaultFunc_NonNilPointer(t *testing.T) {
	v := new("actual")
	called := false
	got := ValOrDefaultFunc(v, func() string {
		called = true
		return "fallback"
	})
	if got != "actual" {
		t.Errorf("expected %q, got %q", "actual", got)
	}
	if called {
		t.Error("default func should not be called when pointer is non-nil")
	}
}

func TestValOrDefaultFunc_NilPointer(t *testing.T) {
	called := false
	got := ValOrDefaultFunc(nil, func() string {
		called = true
		return "computed"
	})
	if got != "computed" {
		t.Errorf("expected %q, got %q", "computed", got)
	}
	if !called {
		t.Error("default func should be called when pointer is nil")
	}
}

func TestValOrDefaultFunc_ZeroValuePointer(t *testing.T) {
	p := new(int)
	got := ValOrDefaultFunc(p, func() int { return 42 })
	if got != 0 {
		t.Errorf("expected 0 (the pointed-to value), got %d", got)
	}
}

func TestValOrDefaultFunc_NilPointerInt(t *testing.T) {
	got := ValOrDefaultFunc(nil, func() int { return 42 })
	if got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}
