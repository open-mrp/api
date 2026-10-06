package versiontransforms

import (
	"testing"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func hiddenRecommendationPayload(t *testing.T) map[string]any {
	t.Helper()
	return hideCosts(marshalResource(t, apiresource.SampleFulfillmentRecommendation), "annual_cogs")
}

func TestFulfillmentRecommendationPreview6To5_HiddenCOGSReadsZero(t *testing.T) {
	t.Parallel()
	tr := &fulfillmentRecommendationForgePreview6To5{}

	result := tr.Transform(constants.ObjectTypeFulfillmentRecommendation, hiddenRecommendationPayload(t))

	assertCostsZero(t, result, "annual_cogs")
	if result["coefficient_of_variation"] != float64(1.8) {
		t.Errorf("the rest of the recommendation must pass through, got %v", result["coefficient_of_variation"])
	}
}

func TestFulfillmentRecommendationPreview6To5_VisibleCOGSIsKept(t *testing.T) {
	t.Parallel()
	tr := &fulfillmentRecommendationForgePreview6To5{}

	result := tr.Transform(constants.ObjectTypeFulfillmentRecommendation, marshalResource(t, apiresource.SampleFulfillmentRecommendation))

	if result["annual_cogs"] != float64(4000) {
		t.Errorf("a caller who may read costs gets the figure, got %v", result["annual_cogs"])
	}
}

// The list and the apply action both answer with a list envelope of recommendations.
func TestFulfillmentRecommendationPreview6To5_WalksListsAndNestedObjects(t *testing.T) {
	t.Parallel()
	tr := &fulfillmentRecommendationForgePreview6To5{}

	result := tr.Transform(constants.ObjectTypeFulfillmentRecommendation, map[string]any{
		"object": "list",
		"data": []any{
			hiddenRecommendationPayload(t),
			map[string]any{"object": "wrapper", "recommendation": hiddenRecommendationPayload(t)},
		},
	})

	rows := result["data"].([]any)
	assertCostsZero(t, rows[0].(map[string]any), "annual_cogs")
	assertCostsZero(t, rows[1].(map[string]any)["recommendation"].(map[string]any), "annual_cogs")
}

func TestFulfillmentRecommendationPreview6To5_MissingCOGSIsNotInvented(t *testing.T) {
	t.Parallel()
	tr := &fulfillmentRecommendationForgePreview6To5{}

	result := tr.Transform(constants.ObjectTypeFulfillmentRecommendation, map[string]any{"object": "list", "data": []any{
		map[string]any{"object": "fulfillment_recommendation"},
	}})

	if _, ok := result["data"].([]any)[0].(map[string]any)["annual_cogs"]; ok {
		t.Error("a payload without the figure must not gain one")
	}
}

func TestFulfillmentRecommendationPreview6To5_LeavesOtherObjectsAlone(t *testing.T) {
	t.Parallel()
	tr := &fulfillmentRecommendationForgePreview6To5{}

	result := tr.Transform(constants.ObjectTypeFulfillmentRecommendation, map[string]any{"object": "item", "annual_cogs": nil})

	if result["annual_cogs"] != nil {
		t.Errorf("only recommendations are downgraded, got %v", result["annual_cogs"])
	}
}

func TestFulfillmentRecommendationPreview6To5_RequestIsIdentity(t *testing.T) {
	t.Parallel()
	tr := &fulfillmentRecommendationForgePreview6To5{}

	got := tr.TransformRequest(constants.ObjectTypeFulfillmentRecommendation, map[string]any{"item_ids": []any{"it_1"}})

	if ids, _ := got["item_ids"].([]any); len(ids) != 1 {
		t.Errorf("the request shape did not change, got %v", got)
	}
}

func TestFulfillmentRecommendationPreview6To5_DefaultRegistryEndToEnd(t *testing.T) {
	t.Parallel()

	for _, to := range []version.APIVersion{version.V1_0_Forge_Preview5, version.V1_0_Forge_Preview1} {
		result := version.Transform(version.V1_0_Forge_Preview6, to, constants.ObjectTypeFulfillmentRecommendation, map[string]any{
			"object": "list", "data": []any{hiddenRecommendationPayload(t)},
		})
		assertCostsZero(t, result["data"].([]any)[0].(map[string]any), "annual_cogs")
	}

	latest := version.Transform(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview6, constants.ObjectTypeFulfillmentRecommendation, hiddenRecommendationPayload(t))
	if latest["annual_cogs"] != nil {
		t.Errorf("preview.6 keeps the null, got %v", latest["annual_cogs"])
	}
}
