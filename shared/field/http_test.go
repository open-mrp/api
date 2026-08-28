package field

import (
	"testing"
)

type OptionalReq struct {
	Name  Optional[string] `json:"name"`
	Phone Optional[string] `json:"phone"`
}

type embeddedOptionalReq struct {
	OptionalReq
	Other Clearable[string] `json:"other,omitzero"`
}

func TestExplicitNullField(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		body     string
		v        any
		wantName string
		wantOK   bool
	}{
		{"explicit null", `{"phone": null}`, &OptionalReq{}, "phone", true},
		{"first null of many", `{"name": null, "phone": null}`, &OptionalReq{}, "name", true},
		{"value not null", `{"phone": "555"}`, &OptionalReq{}, "", false},
		{"absent key", `{}`, &OptionalReq{}, "", false},
		{"embedded field null", `{"name": null}`, &embeddedOptionalReq{}, "name", true},
		{"not an object", `[]`, &OptionalReq{}, "", false},
		{"non-nullable null ignored", `{"other": null}`, &embeddedOptionalReq{}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			name, ok := ExplicitNullField([]byte(tt.body), tt.v)
			if ok != tt.wantOK || name != tt.wantName {
				t.Fatalf("got (%q, %v), want (%q, %v)", name, ok, tt.wantName, tt.wantOK)
			}
		})
	}
}

type untaggedOptionalReq struct {
	Name    Optional[string]
	Skipped Optional[string] `json:"-"`
}

type nestedOptionalReq struct {
	Inner OptionalReq `json:"inner"`
}

// TestExplicitNullField_fallbacks covers the shapes where no field name can be reported, so the
// caller must fall back to a generic 400 rather than naming the wrong parameter.
func TestExplicitNullField_fallbacks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		v    any
	}{
		// encoding/json matches keys case-insensitively and rejects this null, but the lookup
		// here is exact, so the client gets no param.
		{"case-variant key", `{"Name": null}`, &OptionalReq{}},
		{"untagged field", `{"Name": null}`, &untaggedOptionalReq{}},
		{"json dash field", `{"-": null}`, &untaggedOptionalReq{}},
		{"named nested struct", `{"inner": {"name": null}}`, &nestedOptionalReq{}},
		{"nil value", `{"name": null}`, nil},
		{"non-struct value", `{"name": null}`, "not a struct"},
		{"malformed body", `{"name": nul`, &OptionalReq{}},
		{"empty body", ``, &OptionalReq{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if name, ok := ExplicitNullField([]byte(tt.body), tt.v); ok {
				t.Fatalf("want no field, got %q", name)
			}
		})
	}
}

// TestExplicitNullField_shapeComesFromTheType confirms the value is never read: a struct value
// and a typed nil pointer resolve the field just as a populated pointer does.
func TestExplicitNullField_shapeComesFromTheType(t *testing.T) {
	t.Parallel()
	for _, v := range []any{OptionalReq{}, (*OptionalReq)(nil), &OptionalReq{}} {
		name, ok := ExplicitNullField([]byte(`{"phone": null}`), v)
		if !ok || name != "phone" {
			t.Fatalf("%T: got (%q, %v), want (\"phone\", true)", v, name, ok)
		}
	}
}

// TestExplicitNullField_declarationOrderWins pins which of two nulls is named: the first in
// DECLARATION order, not the first in body order (the one the decoder actually failed on).
func TestExplicitNullField_declarationOrderWins(t *testing.T) {
	t.Parallel()
	name, ok := ExplicitNullField([]byte(`{"phone": null, "name": null}`), &OptionalReq{})
	if !ok || name != "name" {
		t.Fatalf("got (%q, %v), want (\"name\", true)", name, ok)
	}
}

// TestExplicitNullField_paddedNull matches the decoder: whitespace around the null still names
// the field.
func TestExplicitNullField_paddedNull(t *testing.T) {
	t.Parallel()
	name, ok := ExplicitNullField([]byte("{\"phone\":\n   null\n}"), &OptionalReq{})
	if !ok || name != "phone" {
		t.Fatalf("got (%q, %v), want (\"phone\", true)", name, ok)
	}
}
