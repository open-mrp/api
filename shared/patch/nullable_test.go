package patch

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestNullable_unsetAndSet(t *testing.T) {
	t.Parallel()

	unset := UnsetNullable[string]()
	if !unset.IsUnset() || unset.IsSet() {
		t.Fatal("expected unset")
	}

	set := SetNullable("x")
	if !set.IsSet() {
		t.Fatal("expected set")
	}
	v, ok := set.Value()
	if !ok || v != "x" {
		t.Fatalf("got %q ok=%v", v, ok)
	}
	if p := set.Ptr(); p == nil || *p != "x" {
		t.Fatalf("Ptr() = %v", p)
	}
}

func TestNullable_marshalJSON(t *testing.T) {
	t.Parallel()

	type payload struct {
		Phone Nullable[string] `json:"phone,omitzero"`
	}

	b, err := json.Marshal(payload{})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "{}" {
		t.Fatalf("expected {}, got %s", b)
	}

	b, err = json.Marshal(payload{Phone: SetNullable("555")})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"phone":"555"}` {
		t.Fatalf("got %s", b)
	}
}

func TestNullable_unmarshalJSON_rejectsNull(t *testing.T) {
	t.Parallel()

	var n Nullable[string]
	err := n.UnmarshalJSON([]byte("null"))
	if !errors.Is(err, ErrExplicitNull) {
		t.Fatalf("expected ErrExplicitNull, got %v", err)
	}
	if !n.IsUnset() {
		t.Fatal("expected still unset after null")
	}
}

func TestNullable_unmarshalJSON_setsValue(t *testing.T) {
	t.Parallel()

	var n Nullable[string]
	if err := n.UnmarshalJSON([]byte(`"hello"`)); err != nil {
		t.Fatal(err)
	}
	if !n.IsSet() {
		t.Fatal("expected set")
	}
	v, _ := n.Value()
	if v != "hello" {
		t.Fatalf("got %q", v)
	}
}

func TestNullable_PtrNullable(t *testing.T) {
	t.Parallel()

	s := "a"
	if !PtrNullable(&s).IsSet() {
		t.Fatal("expected set from pointer")
	}
	if PtrNullable[string](nil).IsSet() {
		t.Fatal("expected unset from nil pointer")
	}
}

func TestIsNullableType(t *testing.T) {
	t.Parallel()

	if !IsNullableType(reflectTypeOf(Nullable[string]{})) {
		t.Fatal("Nullable[string] should be nullable type")
	}
	if IsFieldType(reflectTypeOf(Nullable[string]{})) {
		t.Fatal("Nullable should not be field type")
	}
	if !IsFieldType(reflectTypeOf(Field[string]{})) {
		t.Fatal("Field should be field type")
	}
	if IsNullableType(reflect.TypeFor[*Nullable[string]]()) {
		t.Fatal("*Nullable[string] must not be treated as nullable input type")
	}
}

func reflectTypeOf(v any) reflect.Type {
	return reflect.TypeOf(v)
}
