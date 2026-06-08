package field

import (
	"encoding/json"
	"testing"
)

func TestClearable_threeStates(t *testing.T) {
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

func TestClearable_marshalJSON(t *testing.T) {
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
		Note Clearable[string] `json:"note,omitzero"`
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

func TestClearable_unmarshalJSON(t *testing.T) {
	t.Parallel()

	var unset Clearable[string]
	if !unset.IsUnset() {
		t.Fatal("zero value should be unset")
	}

	var clear Clearable[string]
	if err := clear.UnmarshalJSON([]byte("null")); err != nil {
		t.Fatal(err)
	}
	if !clear.IsClear() {
		t.Fatal("null should clear")
	}

	var set Clearable[string]
	if err := set.UnmarshalJSON([]byte(`"hello"`)); err != nil {
		t.Fatal(err)
	}
	v, ok := set.Value()
	if !ok || v != "hello" {
		t.Fatalf("expected hello, got %q ok=%v", v, ok)
	}
}

type patchValueStruct struct {
	Description Clearable[string] `json:"description,omitzero"`
}

// TestClearableValue_unmarshalJSON verifies that a value Clearable distinguishes all
// three states directly through json.Unmarshal — no repair pass needed. An explicit
// null reaches the addressable field's UnmarshalJSON and is recorded as clear.
func TestClearableValue_unmarshalJSON(t *testing.T) {
	t.Parallel()

	var absent patchValueStruct
	if err := json.Unmarshal([]byte(`{}`), &absent); err != nil {
		t.Fatal(err)
	}
	if !absent.Description.IsUnset() {
		t.Fatal("absent key should be unset")
	}

	var clear patchValueStruct
	if err := json.Unmarshal([]byte(`{"description":null}`), &clear); err != nil {
		t.Fatal(err)
	}
	if !clear.Description.IsClear() {
		t.Fatal("null should clear")
	}

	var set patchValueStruct
	if err := json.Unmarshal([]byte(`{"description":"hello"}`), &set); err != nil {
		t.Fatal(err)
	}
	v, ok := set.Description.Value()
	if !ok || v != "hello" {
		t.Fatalf("expected hello, got %q ok=%v", v, ok)
	}
}
