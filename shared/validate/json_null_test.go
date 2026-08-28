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

// Clearable is exempt from the blank-string check the same way it is exempt from the null check:
// the value reaches the service as a set empty string rather than a 400.
func TestRejectExplicitJSONNulls_allowsBlankForPatchField(t *testing.T) {
	t.Parallel()
	body := []byte(`{"note": ""}`)
	var req patchOptionalPointer
	if err := RejectExplicitJSONNulls(body, &req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRejectExplicitJSONNulls_allowsWhitespaceForPatchField(t *testing.T) {
	t.Parallel()
	body := []byte(`{"note": "   "}`)
	var req patchOptionalPointer
	if err := RejectExplicitJSONNulls(body, &req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplySlicePresenceFlags_nilPointer(t *testing.T) {
	t.Parallel()
	ApplySlicePresenceFlags([]byte(`{"group_ids": []}`), (*patchWithSlice)(nil))
}

func TestApplySlicePresenceFlags_nonStruct(t *testing.T) {
	t.Parallel()
	s := "just a string"
	ApplySlicePresenceFlags([]byte(`{"group_ids": []}`), &s)
	if s != "just a string" {
		t.Fatalf("expected the value to be untouched, got %q", s)
	}
}

func TestApplySlicePresenceFlags_nonObjectBody(t *testing.T) {
	t.Parallel()
	for _, body := range [][]byte{[]byte(`[{"group_ids": []}]`), []byte(`"group_ids"`), []byte(`null`)} {
		var req patchWithSlice
		ApplySlicePresenceFlags(body, &req)
		if req.HasGroupIDs {
			t.Fatalf("expected HasGroupIDs to remain false for %s", body)
		}
	}
}

func TestApplySlicePresenceFlags_malformedBody(t *testing.T) {
	t.Parallel()
	var req patchWithSlice
	ApplySlicePresenceFlags([]byte(`{"group_ids": [`), &req)
	if req.HasGroupIDs {
		t.Fatal("expected HasGroupIDs to remain false for a body that does not parse")
	}
}

type patchRateInput struct {
	Value *string `json:"value,omitempty"`
}

type patchPointerToStruct struct {
	LaborRate *patchRateInput `json:"labor_rate,omitempty"`
}

func TestRejectExplicitJSONNulls_rejectsNullForPointerToStruct(t *testing.T) {
	t.Parallel()
	body := []byte(`{"labor_rate": null}`)
	var req patchPointerToStruct
	err := RejectExplicitJSONNulls(body, &req)
	if err == nil {
		t.Fatal("expected error for labor_rate: null")
	}
	if err.Param != "labor_rate" {
		t.Errorf("expected param 'labor_rate', got: %q", err.Param)
	}
}

func TestRejectExplicitJSONNulls_allowsObjectForPointerToStruct(t *testing.T) {
	t.Parallel()
	body := []byte(`{"labor_rate": {"value": "12.50"}}`)
	var req patchPointerToStruct
	if err := RejectExplicitJSONNulls(body, &req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Production companions are client-visible fields rather than `json:"-"`, so the decoder has
// already filled them by the time this helper runs.
type patchWithPublicSliceFlag struct {
	GroupIDs    []string `json:"group_ids,omitzero"`
	HasGroupIDs bool     `json:"has_group_ids,omitzero"`
}

// A flag set by the client without the slice beside it is the documented way to clear the
// collection, so the helper must leave it alone rather than treat the absent slice as "unchanged".
func TestApplySlicePresenceFlags_keepsClientSetFlagWhenSliceAbsent(t *testing.T) {
	t.Parallel()
	req := patchWithPublicSliceFlag{HasGroupIDs: true}
	ApplySlicePresenceFlags([]byte(`{"has_group_ids": true}`), &req)
	if !req.HasGroupIDs {
		t.Fatal("expected HasGroupIDs to stay true")
	}
}

type patchSliceWithoutCompanion struct {
	GroupIDs []string `json:"group_ids,omitzero"`
}

type patchSliceWithNonBoolCompanion struct {
	GroupIDs    []string `json:"group_ids,omitzero"`
	HasGroupIDs string   `json:"-"`
}

type patchSliceTaggedDash struct {
	GroupIDs    []string `json:"-"`
	HasGroupIDs bool     `json:"-"`
}

// The convention is opt-in: a slice with no bool companion, or one the request does not expose,
// must be skipped rather than reached for by name.
func TestApplySlicePresenceFlags_ignoresSlicesWithoutABoolCompanion(t *testing.T) {
	t.Parallel()

	missing := patchSliceWithoutCompanion{}
	ApplySlicePresenceFlags([]byte(`{"group_ids": []}`), &missing)

	wrongKind := patchSliceWithNonBoolCompanion{}
	ApplySlicePresenceFlags([]byte(`{"group_ids": []}`), &wrongKind)
	if wrongKind.HasGroupIDs != "" {
		t.Errorf("expected the non-bool companion to be untouched, got: %q", wrongKind.HasGroupIDs)
	}

	dashed := patchSliceTaggedDash{}
	ApplySlicePresenceFlags([]byte(`{"group_ids": [], "GroupIDs": []}`), &dashed)
	if dashed.HasGroupIDs {
		t.Error("expected a slice excluded from JSON to have no presence flag")
	}
}
