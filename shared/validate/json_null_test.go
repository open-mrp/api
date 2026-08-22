package validate

import (
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/field"
)

type patchOptionalPointer struct {
	Name             *string                     `json:"name,omitempty"`
	Description      *string                     `json:"description,omitempty"`
	CommissionPolicy *constants.CommissionPolicy `json:"commission_policy,omitempty"`
	Note             field.Clearable[string]     `json:"note,omitzero"`
}

type optionalValueFields struct {
	Name      field.Optional[string]                      `json:"name,omitzero"`
	Status    field.Optional[constants.AccountStatusCode] `json:"status,omitzero"`
	CarrierID field.Optional[string]                      `json:"carrier_id,omitzero"`
}

func TestRejectExplicitJSONNulls_rejectsBlankForOptionalValue(t *testing.T) {
	t.Parallel()
	body := []byte(`{"name": ""}`)
	var req optionalValueFields
	if err := RejectExplicitJSONNulls(body, &req); err == nil {
		t.Fatal("expected error for name: empty string on field.Optional")
	}
}

func TestRejectExplicitJSONNulls_rejectsWhitespaceForOptionalValue(t *testing.T) {
	t.Parallel()
	body := []byte(`{"carrier_id": "   "}`)
	var req optionalValueFields
	if err := RejectExplicitJSONNulls(body, &req); err == nil {
		t.Fatal("expected error for carrier_id: whitespace-only on field.Optional")
	}
}

func TestRejectExplicitJSONNulls_allowsValueAndOmittedForOptionalValue(t *testing.T) {
	t.Parallel()
	for _, body := range [][]byte{[]byte(`{"name": "Acme"}`), []byte(`{}`)} {
		var req optionalValueFields
		if err := RejectExplicitJSONNulls(body, &req); err != nil {
			t.Fatalf("unexpected error for %s: %v", body, err)
		}
	}
}

func TestRejectExplicitJSONNulls_rejectsNullForOptionalPointer(t *testing.T) {
	t.Parallel()
	body := []byte(`{"name": null}`)
	var req patchOptionalPointer
	if err := RejectExplicitJSONNulls(body, &req); err == nil {
		t.Fatal("expected error for name: null")
	}
}

func TestRejectExplicitJSONNulls_allowsOmittedOptionalPointer(t *testing.T) {
	t.Parallel()
	body := []byte(`{}`)
	var req patchOptionalPointer
	if err := RejectExplicitJSONNulls(body, &req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRejectExplicitJSONNulls_allowsValueForOptionalPointer(t *testing.T) {
	t.Parallel()
	body := []byte(`{"name": "Retail"}`)
	var req patchOptionalPointer
	if err := RejectExplicitJSONNulls(body, &req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRejectExplicitJSONNulls_rejectsNullForOptionalPointerWithoutTag(t *testing.T) {
	t.Parallel()
	body := []byte(`{"description": null}`)
	var req patchOptionalPointer
	if err := RejectExplicitJSONNulls(body, &req); err == nil {
		t.Fatal("expected error for description: null")
	}
}

func TestRejectExplicitJSONNulls_allowsNullForPatchField(t *testing.T) {
	t.Parallel()
	body := []byte(`{"note": null}`)
	var req patchOptionalPointer
	if err := RejectExplicitJSONNulls(body, &req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRejectExplicitJSONNulls_rejectsEmptyStringForOptionalPointer(t *testing.T) {
	t.Parallel()
	body := []byte(`{"name": ""}`)
	var req patchOptionalPointer
	if err := RejectExplicitJSONNulls(body, &req); err == nil {
		t.Fatal("expected error for name: empty string")
	}
}

func TestRejectExplicitJSONNulls_rejectsWhitespaceOnlyForOptionalPointer(t *testing.T) {
	t.Parallel()
	body := []byte(`{"name": "   "}`)
	var req patchOptionalPointer
	if err := RejectExplicitJSONNulls(body, &req); err == nil {
		t.Fatal("expected error for name: whitespace-only string")
	}
}

func TestRejectExplicitJSONNulls_commissionPolicyNull(t *testing.T) {
	t.Parallel()
	body := []byte(`{"commission_policy": null}`)
	var req patchOptionalPointer
	if err := RejectExplicitJSONNulls(body, &req); err == nil {
		t.Fatal("expected error")
	}
}

// --- ApplySlicePresenceFlags tests ---

type patchWithSlice struct {
	GroupIDs    []string `json:"group_ids,omitempty"`
	HasGroupIDs bool     `json:"-"`
	Name        *string  `json:"name,omitempty"`
}

func TestApplySlicePresenceFlags_setsHasFlagWhenPresent(t *testing.T) {
	t.Parallel()
	body := []byte(`{"group_ids": ["g1", "g2"]}`)
	var req patchWithSlice
	ApplySlicePresenceFlags(body, &req)
	if !req.HasGroupIDs {
		t.Fatal("expected HasGroupIDs to be true")
	}
}

func TestApplySlicePresenceFlags_setsHasFlagForEmptyArray(t *testing.T) {
	t.Parallel()
	body := []byte(`{"group_ids": []}`)
	var req patchWithSlice
	ApplySlicePresenceFlags(body, &req)
	if !req.HasGroupIDs {
		t.Fatal("expected HasGroupIDs to be true for empty array")
	}
}

func TestApplySlicePresenceFlags_doesNotSetWhenAbsent(t *testing.T) {
	t.Parallel()
	body := []byte(`{"name": "test"}`)
	var req patchWithSlice
	ApplySlicePresenceFlags(body, &req)
	if req.HasGroupIDs {
		t.Fatal("expected HasGroupIDs to remain false")
	}
}

func TestApplySlicePresenceFlags_emptyBody(t *testing.T) {
	t.Parallel()
	body := []byte(`{}`)
	var req patchWithSlice
	ApplySlicePresenceFlags(body, &req)
	if req.HasGroupIDs {
		t.Fatal("expected HasGroupIDs to remain false for empty body")
	}
}
