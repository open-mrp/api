package db

import "testing"

func TestNullStringPtr_NilReturnsInvalid(t *testing.T) {
	t.Parallel()
	got := NullStringPtr(nil)
	if got.Valid {
		t.Fatalf("expected invalid null string for nil input")
	}
}

func TestNullStringPtr_EmptyReturnsInvalid(t *testing.T) {
	t.Parallel()
	empty := ""
	got := NullStringPtr(&empty)
	if got.Valid {
		t.Fatalf("expected invalid null string for empty input")
	}
}

func TestNullStringPtr_NonEmptyReturnsValid(t *testing.T) {
	t.Parallel()
	value := "hello"
	got := NullStringPtr(&value)
	if !got.Valid {
		t.Fatalf("expected valid null string for non-empty input")
	}
	if got.String != "hello" {
		t.Fatalf("expected value %q, got %q", "hello", got.String)
	}
}
