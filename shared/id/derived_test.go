package id

import (
	"regexp"
	"testing"
)

func TestDerivedID(t *testing.T) {
	t.Parallel()
	a := DerivedID(NoteIDPrefix, "item:it_1")
	if a != DerivedID(NoteIDPrefix, "item:it_1") {
		t.Fatal("the same source must give the same ID")
	}
	if a == DerivedID(NoteIDPrefix, "item:it_2") {
		t.Fatal("different sources must give different IDs")
	}
	if !regexp.MustCompile(`^nt_[0-9a-z]{12}$`).MatchString(a) {
		t.Fatalf("%q does not have GenID's shape", a)
	}
}
