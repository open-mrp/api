package versiontransforms

import (
	"reflect"
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func downgradeProductionRunToPreview5(objectType constants.ObjectType, data map[string]any) map[string]any {
	return version.Transform(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview5, objectType, data)
}

func preview6ProductionRun(batchCount any) map[string]any {
	return map[string]any{
		"id":               "prru_1",
		"object":           "production_run",
		"number":           "7",
		"responsible_user": nil,
		"batch_count":      batchCount,
		"batch_summaries":  nil,
		"started_at":       nil,
		"completed_at":     nil,
	}
}

func preview6WeekRelease(run any) map[string]any {
	return map[string]any{
		"object":         "production_schedule_week_release",
		"production_run": run,
		"week_index":     float64(2),
		"batch_count":    nil,
	}
}

func TestProductionRunPreview6To5_WithheldBatchCountReadsZero(t *testing.T) {
	got := downgradeProductionRunToPreview5(constants.ObjectTypeProductionRun, preview6ProductionRun(nil))
	if want := preview6ProductionRun(0); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestProductionRunPreview6To5_VisibleBatchCountIsLeftAlone(t *testing.T) {
	got := downgradeProductionRunToPreview5(constants.ObjectTypeProductionRun, preview6ProductionRun(float64(3)))
	if want := preview6ProductionRun(float64(3)); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestProductionRunPreview6To5_ReachesListsAndEmbeddedRuns(t *testing.T) {
	list := map[string]any{"object": "list", "data": []any{preview6ProductionRun(nil), preview6ProductionRun(float64(3))}}
	got := downgradeProductionRunToPreview5(constants.ObjectTypeProductionRun, list)
	data := got["data"].([]any)
	if data[0].(map[string]any)["batch_count"] != 0 || data[1].(map[string]any)["batch_count"] != float64(3) {
		t.Errorf("list: got %v", data)
	}

	release := downgradeProductionRunToPreview5(constants.ObjectTypeProductionScheduleWeekRelease, preview6WeekRelease(preview6ProductionRun(nil)))
	if run := release["production_run"].(map[string]any); run["batch_count"] != 0 {
		t.Errorf("embedded run: got %v", run)
	}
	if release["batch_count"] != nil {
		t.Errorf("only a production run's batch_count is touched, got %v", release["batch_count"])
	}
}

func TestProductionRunPreview6To5_DataMissing(t *testing.T) {
	for _, in := range []map[string]any{
		preview6WeekRelease(nil),
		{"object": "production_schedule_week_release", "week_index": float64(0)},
		{"object": "production_run", "id": "prru_1", "number": "7"},
	} {
		want := map[string]any{}
		for k, v := range in {
			want[k] = v
		}
		got := downgradeProductionRunToPreview5(constants.ObjectType(in["object"].(string)), in)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestProductionRunPreview6To5_LatestIsUntouched(t *testing.T) {
	got := version.Transform(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview6, constants.ObjectTypeProductionRun, preview6ProductionRun(nil))
	if !reflect.DeepEqual(got, preview6ProductionRun(nil)) {
		t.Errorf("a preview.6 response must pass through, got %v", got)
	}
}

func TestProductionRunPreview6To5_RequestsPassThrough(t *testing.T) {
	got := version.TransformRequest(version.V1_0_Forge_Preview5, version.V1_0_Forge_Preview6, constants.ObjectTypeProductionRun, map[string]any{"number": "8"})
	if !reflect.DeepEqual(got, map[string]any{"number": "8"}) {
		t.Errorf("requests did not change, got %v", got)
	}
}
