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
