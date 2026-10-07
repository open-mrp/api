package versiontransforms

import (
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&fulfillmentRecommendationForgePreview6To5{})
}

// fulfillmentRecommendationForgePreview6To5 downgrades fulfillment recommendations from 1.0.forge-preview.6 to 1.0.forge-preview.5, on both the list and the apply action.
//
// preview.6 made `annual_cogs` nullable: it is null unless the caller holds costs:read. preview.5 typed it as a number, so a preview.5 caller who cannot read costs receives 0 there instead; a caller who can receives the figure in both versions. Requests did not change.
type fulfillmentRecommendationForgePreview6To5 struct{}

var fulfillmentRecommendationCostKeys = map[string][]string{
	string(constants.ObjectTypeFulfillmentRecommendation): {"annual_cogs"},
}

func (t *fulfillmentRecommendationForgePreview6To5) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview6
}

func (t *fulfillmentRecommendationForgePreview6To5) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview5
}

func (t *fulfillmentRecommendationForgePreview6To5) ObjectTypes() []constants.ObjectType {
	return []constants.ObjectType{constants.ObjectTypeFulfillmentRecommendation}
}

func (t *fulfillmentRecommendationForgePreview6To5) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	zeroHiddenCostsIn(data, fulfillmentRecommendationCostKeys)
	return data
}

func (t *fulfillmentRecommendationForgePreview6To5) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}
