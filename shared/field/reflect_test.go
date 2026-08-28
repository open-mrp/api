package field

import (
	"reflect"
	"strings"
	"testing"
)

func TestAssertValuePatchFields_valueOK(t *testing.T) {
	t.Parallel()
	type req struct {
		Name        Optional[string]
		Description Clearable[string]   `json:"description,omitzero"`
		Tags        Clearable[[]string] `json:"tags,omitzero"`
	}
	// Must not panic.
	AssertValuePatchFields(reflect.TypeFor[req]())
	AssertValuePatchFields(reflect.TypeFor[*req]())
}

func TestAssertValuePatchFields_pointerClearablePanics(t *testing.T) {
	t.Parallel()
	type req struct {
		Description *Clearable[string] `json:"description,omitzero"`
	}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for *Clearable field")
		}
		if msg, ok := r.(string); !ok || !strings.Contains(msg, "Description") {
			t.Fatalf("unexpected panic value: %v", r)
		}
	}()
	AssertValuePatchFields(reflect.TypeFor[req]())
}

func TestAssertValuePatchFields_pointerOptionalPanics(t *testing.T) {
	t.Parallel()
	type req struct {
		Name *Optional[string] `json:"name,omitzero"`
	}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for *Optional field")
		}
		if msg, ok := r.(string); !ok || !strings.Contains(msg, "Name") {
			t.Fatalf("unexpected panic value: %v", r)
		}
	}()
	AssertValuePatchFields(reflect.TypeFor[req]())
}

func TestAssertValuePatchFields_embeddedPointerPanics(t *testing.T) {
	t.Parallel()
	type inner struct {
		Note *Clearable[string] `json:"note,omitzero"`
	}
	type req struct {
		inner
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for *Clearable field in embedded struct")
		}
	}()
	AssertValuePatchFields(reflect.TypeFor[req]())
}

// TestAssertValuePatchFields_missingOmitzeroNotChecked records a gap in the assertion: it only
// rejects pointers, so a wrapper declared without omitzero passes startup and fails later, when
// marshaling an unset field errors (see TestClearable_marshalUnset_withoutOmitzero).
func TestAssertValuePatchFields_missingOmitzeroNotChecked(t *testing.T) {
	t.Parallel()
	type req struct {
		Description Clearable[string] `json:"description"`
		Name        Optional[string]  `json:"name"`
	}
	AssertValuePatchFields(reflect.TypeFor[req]())
}

// TestAssertValuePatchFields_namedNestedStructNotChecked pins the recursion boundary: only
// embedded structs are descended into, so a pointer wrapper inside a named sub-struct escapes.
func TestAssertValuePatchFields_namedNestedStructNotChecked(t *testing.T) {
	t.Parallel()
	type inner struct {
		Note *Clearable[string] `json:"note,omitzero"`
	}
	type req struct {
		Inner inner `json:"inner"`
	}
	AssertValuePatchFields(reflect.TypeFor[req]())
}

// TestAssertValuePatchFields_nonStructs must not panic: registration passes whatever a handler
// declares as its request type, including a nil type for a body-less endpoint.
func TestAssertValuePatchFields_nonStructs(t *testing.T) {
	t.Parallel()
	for _, typ := range []reflect.Type{
		nil,
		reflect.TypeFor[string](),
		reflect.TypeFor[[]Clearable[string]](),
		reflect.TypeFor[map[string]Clearable[string]](),
		reflect.TypeFor[*string](),
	} {
		AssertValuePatchFields(typ)
	}
}
