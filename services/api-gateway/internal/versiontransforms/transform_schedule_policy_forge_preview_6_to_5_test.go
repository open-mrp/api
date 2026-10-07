package versiontransforms

import (
	"testing"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func hiddenItemPolicyPayload(t *testing.T) map[string]any {
	t.Helper()
	return hideCosts(marshalResource(t, apiresource.SampleProductionScheduleItemPolicy), schedulePolicyCostKeys...)
}

// hiddenPreviewPayload is a preview whose policies had their costs withheld.
func hiddenPreviewPayload(t *testing.T) map[string]any {
	t.Helper()
	preview := marshalResource(t, apiresource.SampleProductionSchedulePreview)
	for _, row := range previewPolicies(t, preview) {
		hideCosts(row, schedulePolicyCostKeys...)
	}
	return preview
}

func previewPolicies(t *testing.T, preview map[string]any) []map[string]any {
	t.Helper()
	policies, _ := preview["policies"].(map[string]any)
	rows, _ := policies["data"].([]any)
	if len(rows) == 0 {
		t.Fatalf("the preview carries no policies: %v", preview)
	}
	out := make([]map[string]any, len(rows))
	for i, row := range rows {
		out[i] = row.(map[string]any)
	}
	return out
}

func TestSchedulePolicyPreview6To5_PreviewPoliciesReadZero(t *testing.T) {
	t.Parallel()
	tr := &schedulePolicyForgePreview6To5{}

	result := tr.Transform(constants.ObjectTypeProductionSchedulePreview, hiddenPreviewPayload(t))

	for _, policy := range previewPolicies(t, result) {
		assertCostsZero(t, policy, schedulePolicyCostKeys...)
		if policy["eoq_units"] != float64(720) {
			t.Errorf("the rest of the policy must pass through, got eoq_units %v", policy["eoq_units"])
		}
	}
}

func TestSchedulePolicyPreview6To5_ItemPolicyReadsZero(t *testing.T) {
	t.Parallel()
	tr := &schedulePolicyForgePreview6To5{}

	result := tr.Transform(constants.ObjectTypeProductionScheduleItemPolicy, hiddenItemPolicyPayload(t))

	assertCostsZero(t, result, schedulePolicyCostKeys...)
}

func TestSchedulePolicyPreview6To5_VisibleCostsAreKept(t *testing.T) {
	t.Parallel()
	tr := &schedulePolicyForgePreview6To5{}

	item := tr.Transform(constants.ObjectTypeProductionScheduleItemPolicy, marshalResource(t, apiresource.SampleProductionScheduleItemPolicy))
	preview := tr.Transform(constants.ObjectTypeProductionSchedulePreview, marshalResource(t, apiresource.SampleProductionSchedulePreview))

	for _, policy := range append([]map[string]any{item}, previewPolicies(t, preview)...) {
		if policy["unit_cost"] != float64(4) || policy["setup_cost"] != float64(50) || policy["holding_cost"] != float64(1) {
			t.Errorf("a caller who may read costs gets them, got %v / %v / %v", policy["unit_cost"], policy["setup_cost"], policy["holding_cost"])
		}
	}
}

func TestSchedulePolicyPreview6To5_WalksListsAndNestedObjects(t *testing.T) {
	t.Parallel()
	tr := &schedulePolicyForgePreview6To5{}

	result := tr.Transform(constants.ObjectTypeProductionScheduleItemPolicy, map[string]any{
		"object": "list",
		"data": []any{
			hiddenItemPolicyPayload(t),
			map[string]any{"object": "wrapper", "preview": hiddenPreviewPayload(t)},
		},
	})

	rows := result["data"].([]any)
	assertCostsZero(t, rows[0].(map[string]any), schedulePolicyCostKeys...)
	for _, policy := range previewPolicies(t, rows[1].(map[string]any)["preview"].(map[string]any)) {
		assertCostsZero(t, policy, schedulePolicyCostKeys...)
	}
}

// A preview with nothing to plan, or one whose policies list is missing, passes through untouched.
func TestSchedulePolicyPreview6To5_MissingPoliciesAndCostsAreNotInvented(t *testing.T) {
	t.Parallel()
	tr := &schedulePolicyForgePreview6To5{}

	for _, payload := range []map[string]any{
		{"object": "production_schedule_preview"},
		{"object": "production_schedule_preview", "policies": nil},
		{"object": "production_schedule_preview", "policies": map[string]any{"object": "list", "data": []any{}}},
	} {
		result := tr.Transform(constants.ObjectTypeProductionSchedulePreview, payload)
		if result["object"] != "production_schedule_preview" {
			t.Errorf("the preview must pass through, got %v", result)
		}
	}

	result := tr.Transform(constants.ObjectTypeProductionScheduleItemPolicy, map[string]any{"object": "production_schedule_item_policy"})
	for _, key := range schedulePolicyCostKeys {
		if _, ok := result[key]; ok {
			t.Errorf("a policy without %s must not gain it", key)
		}
	}
}

func TestSchedulePolicyPreview6To5_LeavesOtherObjectsAlone(t *testing.T) {
	t.Parallel()
	tr := &schedulePolicyForgePreview6To5{}

	result := tr.Transform(constants.ObjectTypeProductionScheduleItemPolicy, map[string]any{"object": "production_schedule_line", "unit_cost": nil})

	if result["unit_cost"] != nil {
		t.Errorf("only policies are downgraded, got %v", result["unit_cost"])
	}
}

func TestSchedulePolicyPreview6To5_RequestIsIdentity(t *testing.T) {
	t.Parallel()
	tr := &schedulePolicyForgePreview6To5{}

	got := tr.TransformRequest(constants.ObjectTypeProductionSchedulePreview, map[string]any{"horizon_weeks": float64(13)})

	if got["horizon_weeks"] != float64(13) {
		t.Errorf("the request shape did not change, got %v", got)
	}
}

func TestSchedulePolicyPreview6To5_DefaultRegistryEndToEnd(t *testing.T) {
	t.Parallel()

	for _, to := range []version.APIVersion{version.V1_0_Forge_Preview5, version.V1_0_Forge_Preview1} {
		preview := version.Transform(version.V1_0_Forge_Preview6, to, constants.ObjectTypeProductionSchedulePreview, hiddenPreviewPayload(t))
		for _, policy := range previewPolicies(t, preview) {
			assertCostsZero(t, policy, schedulePolicyCostKeys...)
		}

		list := version.Transform(version.V1_0_Forge_Preview6, to, constants.ObjectTypeProductionScheduleItemPolicy, map[string]any{
			"object": "list", "data": []any{hiddenItemPolicyPayload(t)},
		})
		assertCostsZero(t, list["data"].([]any)[0].(map[string]any), schedulePolicyCostKeys...)
	}

	latest := version.Transform(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview6, constants.ObjectTypeProductionScheduleItemPolicy, hiddenItemPolicyPayload(t))
	if latest["unit_cost"] != nil {
		t.Errorf("preview.6 keeps the null, got %v", latest["unit_cost"])
	}
}
