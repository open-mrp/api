package agentep

import (
	"testing"

	"github.com/open-mrp/api/shared/field"
)

// max_steps survives the trip from request to persisted config to response, and stays null when unset.
func TestConfig_MaxStepsRoundTrip(t *testing.T) {
	t.Parallel()
	persisted, err := marshalConfig(ConfigInput{MaxSteps: field.Some(12)})
	if err != nil {
		t.Fatal(err)
	}
	cfg := unmarshalConfig(persisted)
	if cfg.MaxSteps == nil || *cfg.MaxSteps != 12 {
		t.Errorf("max_steps = %v, want 12 (persisted %s)", cfg.MaxSteps, persisted)
	}

	persisted, _ = marshalConfig(ConfigInput{})
	if cfg := unmarshalConfig(persisted); cfg.MaxSteps != nil {
		t.Errorf("unset max_steps should present as null, got %d", *cfg.MaxSteps)
	}
}
