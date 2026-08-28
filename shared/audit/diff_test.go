package audit

import (
	"encoding/json"
	"math"
	"testing"
)

type tagged struct {
	ID   string `audit:"id"`
	Name string `audit:"name"`
	Skip string
}

func TestComputeChanges_ignoresUntaggedFieldsWhenAuto(t *testing.T) {
	t.Parallel()
	old := tagged{ID: "a1", Name: "x", Skip: "secret"}
	newer := tagged{ID: "a1", Name: "y", Skip: "changed"}

	ch := ComputeChanges(old, newer)
	if len(ch) != 1 {
		t.Fatalf("want 1 change (name only), got %d", len(ch))
	}
	if ch[0].Field != "name" {
		t.Fatalf("Field: got %q want name", ch[0].Field)
	}
}

func TestComputeChanges_explicitUntaggedSkipped(t *testing.T) {
	t.Parallel()
	old := tagged{ID: "a1", Name: "x", Skip: "before"}
	newer := tagged{ID: "a1", Name: "x", Skip: "after"}

	ch := ComputeChanges(old, newer, "Skip")
	if len(ch) != 0 {
		t.Fatalf("want no changes for untagged field even when values differ, got %d", len(ch))
	}
}

func TestComputeChanges_createNilOld(t *testing.T) {
	t.Parallel()
	n := tagged{ID: "a1", Name: "n"}

	ch := ComputeChanges(nil, n)
	if len(ch) != 2 {
		t.Fatalf("want 2 tagged fields, got %d", len(ch))
	}
	var sawID, sawName bool
	for _, c := range ch {
		switch c.Field {
		case "id":
			sawID = true
			if string(c.OldValue) != "null" {
				t.Fatalf("id old: %s", c.OldValue)
			}
		case "name":
			sawName = true
		}
	}
	if !sawID || !sawName {
		t.Fatalf("missing keys: id=%v name=%v", sawID, sawName)
	}
}

func TestComputeChanges_deleteNilNew(t *testing.T) {
	t.Parallel()
	o := tagged{ID: "a1", Name: "n"}

	ch := ComputeChanges(o, (*tagged)(nil))
	if len(ch) != 2 {
		t.Fatalf("want 2 tagged fields, got %d", len(ch))
	}
	for _, c := range ch {
		if string(c.NewValue) != "null" {
			t.Fatalf("field %q new should be null, got %s", c.Field, c.NewValue)
		}
		if len(c.OldValue) == 0 || string(c.OldValue) == "null" {
			t.Fatalf("field %q old should be non-null JSON", c.Field)
		}
	}
}

func TestComputeChanges_jsonFragments(t *testing.T) {
	t.Parallel()
	old := tagged{ID: "a1", Name: "a"}
	newer := tagged{ID: "a1", Name: "b"}

	ch := ComputeChanges(old, newer)
	if len(ch) != 1 || ch[0].Field != "name" {
		t.Fatalf("unexpected: %+v", ch)
	}
	var s string
	if err := json.Unmarshal(ch[0].OldValue, &s); err != nil || s != "a" {
		t.Fatalf("old value: %v err=%v", ch[0].OldValue, err)
	}
	if err := json.Unmarshal(ch[0].NewValue, &s); err != nil || s != "b" {
		t.Fatalf("new value: %v err=%v", ch[0].NewValue, err)
	}
}

type embeddedIdentifier struct {
	ID string `audit:"id"`
}

type withEmbedded struct {
	embeddedIdentifier
	Name string `audit:"name"`
}

// A promoted field is addressable by name, so an explicit field list reaches
// audit-tagged fields the embedding struct did not declare itself.
func TestComputeChanges_explicitPromotedField(t *testing.T) {
	t.Parallel()
	old := withEmbedded{embeddedIdentifier{ID: "a1"}, "x"}
	newer := withEmbedded{embeddedIdentifier{ID: "a2"}, "x"}

	ch := ComputeChanges(old, newer, "ID")
	if len(ch) != 1 {
		t.Fatalf("want 1 change, got %d: %+v", len(ch), ch)
	}
	if ch[0].Field != "id" {
		t.Fatalf("Field: got %q want id", ch[0].Field)
	}
	if string(ch[0].OldValue) != `"a1"` || string(ch[0].NewValue) != `"a2"` {
		t.Fatalf("values: old=%s new=%s", ch[0].OldValue, ch[0].NewValue)
	}
}

type numeric struct {
	Amount float64 `audit:"amount"`
}

// A value JSON cannot represent must not abort the mutation it describes: the
// change is still recorded, with a null in place of the value.
func TestComputeChanges_unmarshalableValueBecomesNull(t *testing.T) {
	t.Parallel()
	old := numeric{Amount: 1}
	newer := numeric{Amount: math.NaN()}

	ch := ComputeChanges(old, newer)
	if len(ch) != 1 {
		t.Fatalf("want 1 change, got %d", len(ch))
	}
	if string(ch[0].OldValue) != "1" {
		t.Fatalf("old: got %s want 1", ch[0].OldValue)
	}
	if string(ch[0].NewValue) != "null" {
		t.Fatalf("new: got %s want null", ch[0].NewValue)
	}
}

func TestNewFieldChange_unmarshalableValueBecomesNull(t *testing.T) {
	t.Parallel()

	ch := NewFieldChange("amount", math.Inf(1), 2.5)
	if string(ch.OldValue) != "null" {
		t.Fatalf("old: got %s want null", ch.OldValue)
	}
	if string(ch.NewValue) != "2.5" {
		t.Fatalf("new: got %s want 2.5", ch.NewValue)
	}
}
