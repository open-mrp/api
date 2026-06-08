package field

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestOptional_unsetAndSet(t *testing.T) {
	t.Parallel()

	unset := None[string]()
	if !unset.IsUnset() || unset.IsSet() {
		t.Fatal("expected unset")
	}

	set := Some("x")
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

func TestOptional_marshalJSON(t *testing.T) {
	t.Parallel()

	type payload struct {
		Phone Optional[string] `json:"phone,omitzero"`
	}

	b, err := json.Marshal(payload{})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "{}" {
		t.Fatalf("expected {}, got %s", b)
	}

	b, err = json.Marshal(payload{Phone: Some("555")})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"phone":"555"}` {
		t.Fatalf("got %s", b)
	}
}

func TestOptional_unmarshalJSON_rejectsNull(t *testing.T) {
	t.Parallel()

	var n Optional[string]
	err := n.UnmarshalJSON([]byte("null"))
	if !errors.Is(err, ErrExplicitNull) {
		t.Fatalf("expected ErrExplicitNull, got %v", err)
	}
	if !n.IsUnset() {
		t.Fatal("expected still unset after null")
	}
}

func TestOptional_unmarshalJSON_setsValue(t *testing.T) {
	t.Parallel()

	var n Optional[string]
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

func TestOptional_SomePtr(t *testing.T) {
	t.Parallel()

	s := "a"
	if !SomePtr(&s).IsSet() {
		t.Fatal("expected set from pointer")
	}
	if SomePtr[string](nil).IsSet() {
		t.Fatal("expected unset from nil pointer")
	}
}

func TestIsOptionalType(t *testing.T) {
	t.Parallel()

	if !IsOptionalType(reflectTypeOf(Optional[string]{})) {
		t.Fatal("Optional[string] should be nullable type")
	}
	if IsClearableType(reflectTypeOf(Optional[string]{})) {
		t.Fatal("Nullable should not be field type")
	}
	if !IsClearableType(reflectTypeOf(Clearable[string]{})) {
		t.Fatal("Field should be field type")
	}
	if IsOptionalType(reflect.TypeFor[*Optional[string]]()) {
		t.Fatal("*Optional[string] must not be treated as nullable input type")
	}
}

func reflectTypeOf(v any) reflect.Type {
	return reflect.TypeOf(v)
}
