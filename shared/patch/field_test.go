package patch

import (
	"encoding/json"
	"testing"
)

func TestField_threeStates(t *testing.T) {
	t.Parallel()

	unset := Unset[string]()
	if !unset.IsUnset() || unset.IsClear() || unset.IsSet() {
		t.Fatal("expected unset")
	}

	clear := Clear[string]()
	if !clear.IsClear() || clear.IsUnset() || clear.IsSet() {
		t.Fatal("expected clear")
	}

	set := Set("x")
	if !set.IsSet() {
		t.Fatal("expected set")
	}
	v, ok := set.Value()
	if !ok || v != "x" {
		t.Fatalf("unexpected value: %q ok=%v", v, ok)
	}
}

func TestField_marshalJSON(t *testing.T) {
	t.Parallel()

	set, err := json.Marshal(Set("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if string(set) != `"hello"` {
		t.Fatalf("expected %q, got %s", `"hello"`, set)
	}

	clear, err := json.Marshal(Clear[string]())
	if err != nil {
		t.Fatal(err)
	}
	if string(clear) != "null" {
		t.Fatalf("expected null, got %s", clear)
	}

	type payload struct {
		Note Field[string] `json:"note,omitzero"`
	}
	b, err := json.Marshal(payload{Note: Unset[string]()})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "{}" {
		t.Fatalf("expected empty object, got %s", b)
	}

	b, err = json.Marshal(payload{Note: Set("x")})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"note":"x"}` {
		t.Fatalf("expected set note, got %s", b)
	}
}

func TestField_unmarshalJSON(t *testing.T) {
	t.Parallel()

	var unset Field[string]
	if !unset.IsUnset() {
		t.Fatal("zero value should be unset")
	}

	var clear Field[string]
	if err := clear.UnmarshalJSON([]byte("null")); err != nil {
		t.Fatal(err)
	}
	if !clear.IsClear() {
		t.Fatal("null should clear")
	}

	var set Field[string]
	if err := set.UnmarshalJSON([]byte(`"hello"`)); err != nil {
		t.Fatal(err)
	}
	v, ok := set.Value()
	if !ok || v != "hello" {
		t.Fatalf("expected hello, got %q ok=%v", v, ok)
	}
}

func TestCoalesce_nilIsUnset(t *testing.T) {
	t.Parallel()
	f := Coalesce[string](nil)
	if !f.IsUnset() {
		t.Fatal("expected unset")
	}
}

func TestPtr_roundTrip(t *testing.T) {
	t.Parallel()
	set := Set("x")
	p := new(set)
	if p == nil || !p.IsSet() {
		t.Fatal("expected set")
	}
	if Coalesce(p) != set {
		t.Fatal("Coalesce(Ptr(f)) should equal f")
	}
}

type patchPtrStruct struct {
	Description *Field[string] `json:"description"`
}

func TestFieldPtr_unmarshalJSON(t *testing.T) {
	t.Parallel()

	var absent patchPtrStruct
	if err := json.Unmarshal([]byte(`{}`), &absent); err != nil {
		t.Fatal(err)
	}
	if absent.Description != nil {
		t.Fatal("absent key should leave nil pointer")
	}

	var clear patchPtrStruct
	if err := json.Unmarshal([]byte(`{"description":null}`), &clear); err != nil {
		t.Fatal(err)
	}
	ApplyPtrFieldNulls([]byte(`{"description":null}`), &clear)
	if clear.Description == nil || !clear.Description.IsClear() {
		t.Fatal("null should yield non-nil clear field")
	}

	var set patchPtrStruct
	if err := json.Unmarshal([]byte(`{"description":"hello"}`), &set); err != nil {
		t.Fatal(err)
	}
	ApplyPtrFieldNulls([]byte(`{"description":"hello"}`), &set)
	if set.Description == nil || !set.Description.IsSet() {
		t.Fatal("value should yield non-nil set field")
	}
	v, ok := set.Description.Value()
	if !ok || v != "hello" {
		t.Fatalf("expected hello, got %q ok=%v", v, ok)
	}
}
