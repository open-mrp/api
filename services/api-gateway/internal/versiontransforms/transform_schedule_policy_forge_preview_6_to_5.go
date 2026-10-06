package versiontransforms

import (
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&schedulePolicyForgePreview6To5{})
}

// schedulePolicyForgePreview6To5 downgrades the per-item inventory policies a production schedule is solved with from 1.0.forge-preview.6 to 1.0.forge-preview.5: those a preview returns under `policies`, and those a schedule version persisted, listed as item policies.
//
// preview.6 made `unit_cost`, `setup_cost` and `holding_cost` nullable: they are null unless the caller holds costs:read. preview.5 typed them as numbers, so a preview.5 caller who cannot read costs receives 0 there instead; a caller who can receives the costs in both versions. Requests did not change.
type schedulePolicyForgePreview6To5 struct{}

var schedulePolicyCostKeys = []string{"unit_cost", "setup_cost", "holding_cost"}

func (t *schedulePolicyForgePreview6To5) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview6
}

func (t *schedulePolicyForgePreview6To5) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview5
}

func (t *schedulePolicyForgePreview6To5) ObjectTypes() []constants.ObjectType {
	return []constants.ObjectType{
		constants.ObjectTypeProductionSchedulePreview,
		constants.ObjectTypeProductionScheduleItemPolicy,
	}
}

func (t *schedulePolicyForgePreview6To5) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	zeroSchedulePolicyCostsIn(data)
	return data
}

func (t *schedulePolicyForgePreview6To5) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}

// zeroSchedulePolicyCostsIn finds the policies by their parent, because a preview's policies carry no `object` of their own.
func zeroSchedulePolicyCostsIn(node any) {
	switch v := node.(type) {
	case map[string]any:
		switch asString(v["object"]) {
		case string(constants.ObjectTypeProductionScheduleItemPolicy):
			zeroHiddenCosts(v, schedulePolicyCostKeys...)
		case string(constants.ObjectTypeProductionSchedulePreview):
			policies, _ := v["policies"].(map[string]any)
			rows, _ := policies["data"].([]any)
			for _, row := range rows {
				if policy, ok := row.(map[string]any); ok {
					zeroHiddenCosts(policy, schedulePolicyCostKeys...)
				}
			}
		}
		for _, child := range v {
			zeroSchedulePolicyCostsIn(child)
		}
	case []any:
		for _, child := range v {
			zeroSchedulePolicyCostsIn(child)
		}
	}
}
