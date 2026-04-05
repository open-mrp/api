package textutil

import "testing"

func TestPluralize_Int(t *testing.T) {
	t.Parallel()
	if got := Pluralize(1, "record", "records"); got != "record" {
		t.Fatalf("expected singular for 1, got %q", got)
	}
	if got := Pluralize(2, "record", "records"); got != "records" {
		t.Fatalf("expected plural for >1, got %q", got)
	}
	if got := Pluralize(0, "record", "records"); got != "records" {
		t.Fatalf("expected plural for 0, got %q", got)
	}
}
