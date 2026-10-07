package versiontransforms

import "testing"

// hideCosts nulls keys on obj, as the gateway does for a caller without costs:read.
func hideCosts(obj map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		obj[key] = nil
	}
	return obj
}

func assertCostsZero(t *testing.T, obj map[string]any, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if got, ok := obj[key]; !ok || got != float64(0) {
			t.Errorf("%s: got %v (present %v), want 0", key, got, ok)
		}
	}
}

func TestZeroHiddenCosts_KeepsAVisibleValueAndInventsNoKey(t *testing.T) {
	t.Parallel()
	obj := map[string]any{"unit_cost": float64(4), "setup_cost": nil}

	zeroHiddenCosts(obj, "unit_cost", "setup_cost", "holding_cost")

	if obj["unit_cost"] != float64(4) {
		t.Errorf("a cost the caller may read must pass through, got %v", obj["unit_cost"])
	}
	if obj["setup_cost"] != float64(0) {
		t.Errorf("a withheld cost must read 0, got %v", obj["setup_cost"])
	}
	if _, ok := obj["holding_cost"]; ok {
		t.Error("a key the payload did not carry must not be added")
	}
}
