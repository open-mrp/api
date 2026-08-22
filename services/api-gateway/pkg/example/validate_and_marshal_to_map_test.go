package apiexample

import (
	"testing"
	"time"

	"github.com/open-mrp/api/shared/field"
)

type updateBodyExample struct {
	Note         *string `json:"note,omitempty"`
	HasBeenSent  *bool   `json:"has_been_sent,omitempty"`
	IsEdiSent    *bool   `json:"is_edi_sent,omitempty"`
	IsPaidInFull *bool   `json:"is_paid_in_full,omitempty"`
}

func TestValidateAndMarshalToMapOmitsUnsetNilPointers(t *testing.T) {
	t.Parallel()

	note := "Payment received via wire transfer"
	hasBeenSent := true
	m := ValidateAndMarshalToMap(&updateBodyExample{
		Note:        &note,
		HasBeenSent: &hasBeenSent,
	})

	if _, ok := m["is_edi_sent"]; ok {
		t.Fatalf("unset *bool should be omitted, got %#v", m["is_edi_sent"])
	}
	if _, ok := m["is_paid_in_full"]; ok {
		t.Fatalf("unset *bool should be omitted, got %#v", m["is_paid_in_full"])
	}
	if m["note"] != note {
		t.Fatalf("note = %#v", m["note"])
	}
	if m["has_been_sent"] != hasBeenSent {
		t.Fatalf("has_been_sent = %#v", m["has_been_sent"])
	}
}

type nestedExample struct {
	ID string `json:"id"`
}

// resourceExample mirrors a response resource: nullable fields are plain
// pointers without omitempty, so a nil value is a "value or null" field that
// must appear in the example as null rather than being dropped.
type resourceExample struct {
	ID        string         `json:"id"`
	FlatRate  *nestedExample `json:"flat_rate"`
	Minimum   *string        `json:"minimum_order_value"`
	Tags      []string       `json:"tags"`
	CreatedAt time.Time      `json:"created_at"`
}

func TestValidateAndMarshalToMapEmitsNullForNullableResponseFields(t *testing.T) {
	t.Parallel()

	m := ValidateAndMarshalToMap(&resourceExample{
		ID:        "shtm_123",
		FlatRate:  nil,
		Minimum:   nil,
		CreatedAt: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
	})

	for _, key := range []string{"flat_rate", "minimum_order_value"} {
		val, ok := m[key]
		if !ok {
			t.Fatalf("nullable field %q must be present in the example", key)
		}
		if val != nil {
			t.Fatalf("nullable field %q should be null, got %#v", key, val)
		}
	}
	if m["created_at"] != "2026-05-10T00:00:00Z" {
		t.Fatalf("created_at = %#v", m["created_at"])
	}
	// A nil slice is a non-nullable array on the wire's schema, so it must render
	// as [] (present, empty) — never null and never omitted.
	tags, ok := m["tags"]
	if !ok {
		t.Fatalf("non-nullable slice field must be present in the example")
	}
	arr, isArr := tags.([]any)
	if !isArr || len(arr) != 0 {
		t.Fatalf("nil slice should render as empty array, got %#v", tags)
	}
}

type patchBodyExample struct {
	// Path-only field: no json tag, must never appear in the body example.
	CustomerID string `path:"id"`

	Name  field.Optional[string]  `json:"name,omitzero"`
	Email field.Clearable[string] `json:"email,omitzero"`
	Phone field.Clearable[string] `json:"phone,omitzero"`
	Note  field.Optional[string]  `json:"note,omitzero"`
}

func TestValidateAndMarshalToMapUnwrapsPatchFields(t *testing.T) {
	t.Parallel()

	m := ValidateAndMarshalToMap(&patchBodyExample{
		CustomerID: "cus_123",
		Name:       field.Some("Acme Inc."),
		Email:      field.Clear[string](), // explicit clear -> null
		Phone:      field.Unset[string](), // unset -> omitted
		Note:       field.None[string](),  // unset -> omitted
	})

	if _, ok := m["CustomerID"]; ok {
		t.Fatalf("path-only field must not appear in body example, got %#v", m["CustomerID"])
	}
	if m["name"] != "Acme Inc." {
		t.Fatalf("set Optional should encode its value, name = %#v", m["name"])
	}
	if val, ok := m["email"]; !ok || val != nil {
		t.Fatalf("cleared Clearable should be null, email present=%v value=%#v", ok, val)
	}
	if _, ok := m["phone"]; ok {
		t.Fatalf("unset Clearable should be omitted, got %#v", m["phone"])
	}
	if _, ok := m["note"]; ok {
		t.Fatalf("unset Optional should be omitted, got %#v", m["note"])
	}
}
