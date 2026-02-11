package ptrutil

import "testing"

// --- Ptr ---

func TestPtr_String(t *testing.T) {
	p := Ptr("hello")
	if p == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *p != "hello" {
		t.Errorf("expected %q, got %q", "hello", *p)
	}
}

func TestPtr_Int(t *testing.T) {
	p := Ptr(42)
	if *p != 42 {
		t.Errorf("expected 42, got %d", *p)
	}
}

func TestPtr_Bool(t *testing.T) {
	p := Ptr(true)
	if *p != true {
		t.Errorf("expected true, got %v", *p)
	}
}

func TestPtr_ZeroValue(t *testing.T) {
	p := Ptr("")
	if p == nil {
		t.Fatal("expected non-nil pointer even for zero value")
	}
	if *p != "" {
		t.Errorf("expected empty string, got %q", *p)
	}
}

func TestPtr_Struct(t *testing.T) {
	type S struct{ X int }
	p := Ptr(S{X: 7})
	if p.X != 7 {
		t.Errorf("expected X=7, got %d", p.X)
	}
}

func TestPtr_ReturnsDistinctPointers(t *testing.T) {
	a := Ptr(1)
	b := Ptr(1)
	if a == b {
		t.Error("expected distinct pointers for separate calls")
	}
}

func TestPtr_MutationDoesNotAffectOriginal(t *testing.T) {
	v := 10
	p := Ptr(v)
	*p = 20
	if v != 10 {
		t.Error("mutating the pointer should not affect the original value")
	}
}

// --- ValOrDefault ---

func TestValOrDefault_NonNilPointer(t *testing.T) {
	v := Ptr("actual")
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
	p := Ptr(0)
	got := ValOrDefault(p, 99)
	if got != 0 {
		t.Errorf("expected 0 (the pointed-to value), got %d", got)
	}
}

func TestValOrDefault_EmptyStringPointer(t *testing.T) {
	p := Ptr("")
	got := ValOrDefault(p, "default")
	if got != "" {
		t.Errorf("expected empty string (the pointed-to value), got %q", got)
	}
}

func TestValOrDefault_FalsePointer(t *testing.T) {
	p := Ptr(false)
	got := ValOrDefault(p, true)
	if got != false {
		t.Errorf("expected false (the pointed-to value), got %v", got)
	}
}

// --- ValOrDefaultFunc ---

func TestValOrDefaultFunc_NonNilPointer(t *testing.T) {
	v := Ptr("actual")
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
	p := Ptr(0)
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
