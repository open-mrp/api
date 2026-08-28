package ptrutil

import (
	"testing"
	"time"
)

// --- ValOrDefault ---

func TestValOrDefault_NonNilPointer(t *testing.T) {
	t.Parallel()
	v := new("actual")
	got := ValOrDefault(v, "fallback")
	if got != "actual" {
		t.Errorf("expected %q, got %q", "actual", got)
	}
}

func TestValOrDefault_NilPointer(t *testing.T) {
	t.Parallel()
	got := ValOrDefault(nil, "fallback")
	if got != "fallback" {
		t.Errorf("expected %q, got %q", "fallback", got)
	}
}

func TestValOrDefault_NilPointerInt(t *testing.T) {
	t.Parallel()
	got := ValOrDefault(nil, 99)
	if got != 99 {
		t.Errorf("expected 99, got %d", got)
	}
}

func TestValOrDefault_ZeroValuePointer(t *testing.T) {
	t.Parallel()
	p := new(0)
	got := ValOrDefault(p, 99)
	if got != 0 {
		t.Errorf("expected 0 (the pointed-to value), got %d", got)
	}
}

func TestValOrDefault_EmptyStringPointer(t *testing.T) {
	t.Parallel()
	p := new("")
	got := ValOrDefault(p, "default")
	if got != "" {
		t.Errorf("expected empty string (the pointed-to value), got %q", got)
	}
}

func TestValOrDefault_FalsePointer(t *testing.T) {
	t.Parallel()
	p := new(false)
	got := ValOrDefault(p, true)
	if got != false {
		t.Errorf("expected false (the pointed-to value), got %v", got)
	}
}

// --- ValOrDefaultFunc ---

func TestValOrDefaultFunc_NonNilPointer(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	p := new(int)
	got := ValOrDefaultFunc(p, func() int { return 42 })
	if got != 0 {
		t.Errorf("expected 0 (the pointed-to value), got %d", got)
	}
}

func TestValOrDefaultFunc_NilPointerInt(t *testing.T) {
	t.Parallel()
	got := ValOrDefaultFunc(nil, func() int { return 42 })
	if got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

// --- Deref ---

func TestDeref_NonNilPointer(t *testing.T) {
	t.Parallel()
	p := new("actual")
	if got := Deref(p); got != "actual" {
		t.Errorf("expected %q, got %q", "actual", got)
	}
}

func TestDeref_NilPointerString(t *testing.T) {
	t.Parallel()
	if got := Deref[string](nil); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestDeref_NilPointerInt(t *testing.T) {
	t.Parallel()
	if got := Deref[int](nil); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestDeref_NilPointerBool(t *testing.T) {
	t.Parallel()
	if got := Deref[bool](nil); got != false {
		t.Errorf("expected false, got %v", got)
	}
}

func TestDeref_ZeroValuePointer(t *testing.T) {
	t.Parallel()
	p := new(0)
	if got := Deref(p); got != 0 {
		t.Errorf("expected 0 (the pointed-to value), got %d", got)
	}
}

func TestDeref_NilTimePointer(t *testing.T) {
	t.Parallel()
	got := Deref[time.Time](nil)
	if !got.IsZero() {
		t.Errorf("expected the zero time, got %v", got)
	}
}

func TestDeref_NonNilTimePointer(t *testing.T) {
	t.Parallel()
	want := time.Date(2024, time.March, 1, 12, 0, 0, 0, time.UTC)
	if got := Deref(&want); !got.Equal(want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestDeref_NilStructPointer(t *testing.T) {
	t.Parallel()
	type record struct {
		Name  string
		Count int
	}
	got := Deref[record](nil)
	if got != (record{}) {
		t.Errorf("expected the zero struct, got %+v", got)
	}
}

func TestDeref_NilSlicePointer(t *testing.T) {
	t.Parallel()
	got := Deref[[]string](nil)
	if got != nil {
		t.Errorf("expected a nil slice, got %v", got)
	}
	if len(got) != 0 {
		t.Errorf("expected an empty slice, got %d elements", len(got))
	}
}

func TestDeref_NilMapPointer(t *testing.T) {
	t.Parallel()
	got := Deref[map[string]int](nil)
	if got != nil {
		t.Errorf("expected a nil map, got %v", got)
	}
}

// --- ApplyIfSet ---

func TestApplyIfSet_NilSrcLeavesNonZeroDstUntouched(t *testing.T) {
	t.Parallel()
	dst := "existing"
	ApplyIfSet(&dst, nil)
	if dst != "existing" {
		t.Errorf("expected %q to survive an omitted field, got %q", "existing", dst)
	}
}

func TestApplyIfSet_NilSrcLeavesZeroDstUntouched(t *testing.T) {
	t.Parallel()
	dst := 0
	ApplyIfSet(&dst, nil)
	if dst != 0 {
		t.Errorf("expected 0, got %d", dst)
	}
}

func TestApplyIfSet_NonNilSrcOverwrites(t *testing.T) {
	t.Parallel()
	dst := "existing"
	ApplyIfSet(&dst, new("patched"))
	if dst != "patched" {
		t.Errorf("expected %q, got %q", "patched", dst)
	}
}

// A field explicitly set to its zero value is a clear, not an omission.
func TestApplyIfSet_ZeroValueSrcOverwrites(t *testing.T) {
	t.Parallel()
	dst := "existing"
	ApplyIfSet(&dst, new(""))
	if dst != "" {
		t.Errorf("expected an explicit empty string to overwrite, got %q", dst)
	}

	count := 42
	ApplyIfSet(&count, new(0))
	if count != 0 {
		t.Errorf("expected an explicit 0 to overwrite, got %d", count)
	}

	flag := true
	ApplyIfSet(&flag, new(false))
	if flag != false {
		t.Errorf("expected an explicit false to overwrite, got %v", flag)
	}
}

func TestApplyIfSet_StructValue(t *testing.T) {
	t.Parallel()
	type addr struct {
		City string
		Zip  string
	}
	dst := addr{City: "Denver", Zip: "80202"}
	ApplyIfSet(&dst, &addr{City: "Boise", Zip: "83702"})
	if dst != (addr{City: "Boise", Zip: "83702"}) {
		t.Errorf("expected the src struct, got %+v", dst)
	}
	ApplyIfSet(&dst, nil)
	if dst != (addr{City: "Boise", Zip: "83702"}) {
		t.Errorf("expected a nil src to leave the struct untouched, got %+v", dst)
	}
}

func TestApplyIfSet_NilDstWithNilSrcIsNoop(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("expected no panic when both dst and src are nil, got %v", r)
		}
	}()
	ApplyIfSet[string](nil, nil)
}

// A nil dst is a programmer error, not an input case — pinned so the panic is not mistaken for a silent no-op.
func TestApplyIfSet_NilDstWithNonNilSrcPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected a panic when dst is nil and src is set")
		}
	}()
	ApplyIfSet(nil, new("patched"))
}
