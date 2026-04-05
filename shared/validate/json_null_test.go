package validate

import (
	"testing"

	"github.com/augno/api/shared/constants"
)

type patchNullableFalse struct {
	Name             *string                     `json:"name,omitempty" nullable:"false"`
	Description      *string                     `json:"description,omitempty"`
	CommissionPolicy *constants.CommissionPolicy `json:"commission_policy,omitempty" nullable:"false"`
}

func TestRejectExplicitJSONNulls_rejectsNullForNullableFalse(t *testing.T) {
	t.Parallel()
	body := []byte(`{"name": null}`)
	var req patchNullableFalse
	if err := RejectExplicitJSONNulls(body, &req); err == nil {
		t.Fatal("expected error for name: null")
	}
}

func TestRejectExplicitJSONNulls_allowsOmittedNullableFalse(t *testing.T) {
	t.Parallel()
	body := []byte(`{}`)
	var req patchNullableFalse
	if err := RejectExplicitJSONNulls(body, &req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRejectExplicitJSONNulls_allowsValueForNullableFalse(t *testing.T) {
	t.Parallel()
	body := []byte(`{"name": "Retail"}`)
	var req patchNullableFalse
	if err := RejectExplicitJSONNulls(body, &req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRejectExplicitJSONNulls_allowsNullForUntaggedField(t *testing.T) {
	t.Parallel()
	body := []byte(`{"description": null}`)
	var req patchNullableFalse
	if err := RejectExplicitJSONNulls(body, &req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRejectExplicitJSONNulls_commissionPolicyNull(t *testing.T) {
	t.Parallel()
	body := []byte(`{"commission_policy": null}`)
	var req patchNullableFalse
	if err := RejectExplicitJSONNulls(body, &req); err == nil {
		t.Fatal("expected error")
	}
}

// --- ApplyExplicitNulls tests ---

type PatchNullableTrue struct {
	CarrierID *string `json:"carrier_id,omitempty" nullable:"true"`
	Name      *string `json:"name,omitempty"`
	PolicyID  *string `json:"policy_id,omitempty" nullable:"false"`
	Count     *int    `json:"count,omitempty" nullable:"true"`
}

type patchNullableTrue = PatchNullableTrue

func TestApplyExplicitNulls_setsEmptyStringForExplicitNull(t *testing.T) {
	t.Parallel()
	body := []byte(`{"carrier_id": null}`)
	var req patchNullableTrue
	ApplyExplicitNulls(body, &req)
	if req.CarrierID == nil {
		t.Fatal("expected CarrierID to be set to empty string, got nil")
	}
	if *req.CarrierID != "" {
		t.Fatalf("expected empty string, got %q", *req.CarrierID)
	}
}

func TestApplyExplicitNulls_leavesAbsentFieldNil(t *testing.T) {
	t.Parallel()
	body := []byte(`{}`)
	var req patchNullableTrue
	ApplyExplicitNulls(body, &req)
	if req.CarrierID != nil {
		t.Fatalf("expected CarrierID to remain nil, got %q", *req.CarrierID)
	}
}

func TestApplyExplicitNulls_preservesProvidedValue(t *testing.T) {
	t.Parallel()
	body := []byte(`{"carrier_id": "cr_123"}`)
	val := "cr_123"
	req := patchNullableTrue{CarrierID: &val}
	ApplyExplicitNulls(body, &req)
	if req.CarrierID == nil || *req.CarrierID != "cr_123" {
		t.Fatalf("expected cr_123, got %v", req.CarrierID)
	}
}

func TestApplyExplicitNulls_doesNotTouchUntaggedField(t *testing.T) {
	t.Parallel()
	body := []byte(`{"name": null}`)
	var req patchNullableTrue
	ApplyExplicitNulls(body, &req)
	if req.Name != nil {
		t.Fatalf("expected Name to remain nil, got %q", *req.Name)
	}
}

func TestApplyExplicitNulls_doesNotTouchNullableFalseField(t *testing.T) {
	t.Parallel()
	body := []byte(`{"policy_id": null}`)
	var req patchNullableTrue
	ApplyExplicitNulls(body, &req)
	if req.PolicyID != nil {
		t.Fatalf("expected PolicyID to remain nil, got %q", *req.PolicyID)
	}
}

func TestApplyExplicitNulls_ignoresNonStringPointerTypes(t *testing.T) {
	t.Parallel()
	body := []byte(`{"count": null}`)
	var req patchNullableTrue
	ApplyExplicitNulls(body, &req)
	if req.Count != nil {
		t.Fatal("expected Count to remain nil")
	}
}

type patchNullableEmbedded struct {
	PatchNullableTrue
	ExtraID *string `json:"extra_id,omitempty" nullable:"true"`
}

func TestApplyExplicitNulls_handlesEmbeddedStructs(t *testing.T) {
	t.Parallel()
	body := []byte(`{"carrier_id": null, "extra_id": null}`)
	var req patchNullableEmbedded
	ApplyExplicitNulls(body, &req)
	if req.CarrierID == nil || *req.CarrierID != "" {
		t.Fatal("expected embedded CarrierID to be empty string")
	}
	if req.ExtraID == nil || *req.ExtraID != "" {
		t.Fatal("expected ExtraID to be empty string")
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
