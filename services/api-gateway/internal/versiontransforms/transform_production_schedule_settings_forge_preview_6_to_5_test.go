package versiontransforms

import (
	"testing"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func hiddenSettingsPayload(t *testing.T) map[string]any {
	t.Helper()
	return hideCosts(marshalResource(t, apiresource.SampleProductionScheduleSettings), "changeover_labor_rate")
}

func TestProductionScheduleSettingsPreview6To5_HiddenRateReadsZero(t *testing.T) {
	t.Parallel()
	tr := &productionScheduleSettingsForgePreview6To5{}

	result := tr.Transform(constants.ObjectTypeProductionScheduleSettings, hiddenSettingsPayload(t))

	assertCostsZero(t, result, "changeover_labor_rate")
	if result["holding_rate_pct"] != float64(0.25) {
		t.Errorf("the rest of the settings must pass through, got holding_rate_pct %v", result["holding_rate_pct"])
	}
}

func TestProductionScheduleSettingsPreview6To5_VisibleRateIsKept(t *testing.T) {
	t.Parallel()
	tr := &productionScheduleSettingsForgePreview6To5{}

	result := tr.Transform(constants.ObjectTypeProductionScheduleSettings, marshalResource(t, apiresource.SampleProductionScheduleSettings))

	if result["changeover_labor_rate"] != float64(20) {
		t.Errorf("a caller who may read costs gets the rate, got %v", result["changeover_labor_rate"])
	}
}

func TestProductionScheduleSettingsPreview6To5_WalksListsAndNestedObjects(t *testing.T) {
	t.Parallel()
	tr := &productionScheduleSettingsForgePreview6To5{}

	result := tr.Transform(constants.ObjectTypeProductionScheduleSettings, map[string]any{
		"object": "list",
		"data": []any{
			map[string]any{"object": "production_schedule_settings", "changeover_labor_rate": nil},
			map[string]any{"object": "wrapper", "settings": map[string]any{"object": "production_schedule_settings", "changeover_labor_rate": nil}},
		},
	})

	rows := result["data"].([]any)
	assertCostsZero(t, rows[0].(map[string]any), "changeover_labor_rate")
	assertCostsZero(t, rows[1].(map[string]any)["settings"].(map[string]any), "changeover_labor_rate")
}

func TestProductionScheduleSettingsPreview6To5_MissingRateIsNotInvented(t *testing.T) {
	t.Parallel()
	tr := &productionScheduleSettingsForgePreview6To5{}

	result := tr.Transform(constants.ObjectTypeProductionScheduleSettings, map[string]any{"object": "production_schedule_settings"})

	if _, ok := result["changeover_labor_rate"]; ok {
		t.Error("a payload without the rate must not gain one")
	}
}

func TestProductionScheduleSettingsPreview6To5_LeavesOtherObjectsAlone(t *testing.T) {
	t.Parallel()
	tr := &productionScheduleSettingsForgePreview6To5{}

	result := tr.Transform(constants.ObjectTypeProductionScheduleSettings, map[string]any{"object": "production_schedule", "changeover_labor_rate": nil})

	if result["changeover_labor_rate"] != nil {
		t.Errorf("only the settings resource is downgraded, got %v", result["changeover_labor_rate"])
	}
}

func TestProductionScheduleSettingsPreview6To5_RequestIsIdentity(t *testing.T) {
	t.Parallel()
	tr := &productionScheduleSettingsForgePreview6To5{}

	got := tr.TransformRequest(constants.ObjectTypeProductionScheduleSettings, map[string]any{"changeover_labor_rate": float64(41.5)})

	if got["changeover_labor_rate"] != float64(41.5) {
		t.Errorf("the request shape did not change, got %v", got)
	}
}

func TestProductionScheduleSettingsPreview6To5_DefaultRegistryEndToEnd(t *testing.T) {
	t.Parallel()

	for _, to := range []version.APIVersion{version.V1_0_Forge_Preview5, version.V1_0_Forge_Preview1} {
		result := version.Transform(version.V1_0_Forge_Preview6, to, constants.ObjectTypeProductionScheduleSettings, hiddenSettingsPayload(t))
		assertCostsZero(t, result, "changeover_labor_rate")
	}

	latest := version.Transform(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview6, constants.ObjectTypeProductionScheduleSettings, hiddenSettingsPayload(t))
	if latest["changeover_labor_rate"] != nil {
		t.Errorf("preview.6 keeps the null, got %v", latest["changeover_labor_rate"])
	}
}
