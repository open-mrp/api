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

func TestClearable_BackfillUnsetPtr(t *testing.T) {
	t.Parallel()

	existing := "existing"
	tests := []struct {
		name      string
		in        Clearable[string]
		existing  *string
		wantState string
		wantValue string
	}{
		{"unset with existing takes the existing value", Unset[string](), &existing, "set", "existing"},
		// A nil existing means the column is already NULL, so the field stays unset and the
		// sql helpers write NULL — the same value the row already holds.
		{"unset with nil existing stays unset", Unset[string](), nil, "unset", ""},
		{"clear is not backfilled", Clear[string](), &existing, "clear", ""},
		{"set keeps the requested value", Set("new"), &existing, "set", "new"},
		{"set blank keeps the blank", Set(""), &existing, "set", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.in.BackfillUnsetPtr(tt.existing)
			if state := stateName(got); state != tt.wantState {
				t.Fatalf("state: got %s, want %s", state, tt.wantState)
			}
			if v, _ := got.Value(); v != tt.wantValue {
				t.Fatalf("value: got %q, want %q", v, tt.wantValue)
			}
		})
	}
}

// TestClearable_BackfillUnsetPtr_nilExistingWritesNull ties the unset+nil case to what the
// repository layer does with it: NULL, which is why callers must backfill before the sql helpers.
func TestClearable_BackfillUnsetPtr_nilExistingWritesNull(t *testing.T) {
	t.Parallel()

	got := StringToNullString(Unset[string]().BackfillUnsetPtr(nil))
	if got.Valid {
		t.Fatalf("want NULL, got %+v", got)
	}
}

func TestClearable_StringPtrAfterBackfill(t *testing.T) {
	t.Parallel()

	existing := "existing"
	tests := []struct {
		name     string
		in       Clearable[string]
		existing *string
		want     *string
	}{
		{"unset takes the existing value", Unset[string](), &existing, &existing},
		{"unset with nil existing stays nil", Unset[string](), nil, nil},
		{"clear beats the existing value", Clear[string](), &existing, nil},
		{"set wins", Set("new"), &existing, ptrTo("new")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.in.StringPtrAfterBackfill(tt.existing)
			switch {
			case tt.want == nil && got != nil:
				t.Fatalf("want nil, got %q", *got)
			case tt.want != nil && got == nil:
				t.Fatalf("want %q, got nil", *tt.want)
			case tt.want != nil && *got != *tt.want:
				t.Fatalf("got %q, want %q", *got, *tt.want)
			}
		})
	}
}

// TestClearable_StringPtrAfterBackfill_anyInnerType documents the name: the method's "string" is
// a type-parameter name, not the string type, so it works on every Clearable[T].
func TestClearable_StringPtrAfterBackfill_anyInnerType(t *testing.T) {
	t.Parallel()

	existing := 7
	got := Unset[int]().StringPtrAfterBackfill(&existing)
	if got == nil || *got != 7 {
		t.Fatalf("got %v, want a pointer to 7", got)
	}
	if p := Clear[int]().StringPtrAfterBackfill(&existing); p != nil {
		t.Fatalf("clear must yield nil, got %d", *p)
	}
}

func stateName[T any](f Clearable[T]) string {
	switch {
	case f.IsUnset():
		return "unset"
	case f.IsClear():
		return "clear"
	default:
		return "set"
	}
}

func ptrTo[T any](v T) *T { return &v }
