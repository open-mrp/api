package apiexample

import "testing"

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
