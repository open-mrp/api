package versiontransforms

import (
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&productionRunForgePreview6To5{})
}

// productionRunForgePreview6To5 bridges production runs from 1.0.forge-preview.6 to 1.0.forge-preview.5.
//
// preview.6 withholds a run's batch count, like its batch summaries and responsible user, from anyone outside the seller's account, so `batch_count` became nullable where preview.5 always carried a number. A withheld count reads 0 on preview.5, as a withheld cost figure does.
type productionRunForgePreview6To5 struct{}

func (t *productionRunForgePreview6To5) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview6
}

func (t *productionRunForgePreview6To5) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview5
}

func (t *productionRunForgePreview6To5) ObjectTypes() []constants.ObjectType {
	return []constants.ObjectType{
		constants.ObjectTypeProductionRun,
		constants.ObjectTypeProductionScheduleWeekRelease,
	}
}

func (t *productionRunForgePreview6To5) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	zeroWithheldBatchCounts(data)
	return data
}

func (t *productionRunForgePreview6To5) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	// Request shapes did not change between preview.5 and preview.6.
	return data
}

// zeroWithheldBatchCounts sets, in place, the batch count of every production run in node whose count was withheld to 0.
func zeroWithheldBatchCounts(node any) {
	switch v := node.(type) {
	case map[string]any:
		if v["object"] == string(constants.ObjectTypeProductionRun) {
			if count, present := v["batch_count"]; present && count == nil {
				v["batch_count"] = 0
			}
		}
		for _, child := range v {
			zeroWithheldBatchCounts(child)
		}
	case []any:
		for _, child := range v {
			zeroWithheldBatchCounts(child)
		}
	}
}
