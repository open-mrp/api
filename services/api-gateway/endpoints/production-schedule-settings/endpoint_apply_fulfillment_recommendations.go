package productionschedulesettingsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to adopt fulfillment recommendations for specific items.
type ApplyFulfillmentRecommendationsRequest struct {
	// Items whose recommendation should be adopted.
	//
	// Named explicitly rather than applied wholesale: adopting advice in bulk without saying what is being adopted is how a plant changes what it builds by accident. Items not named here are left exactly as they are.
	ItemIDs []string `json:"item_ids" validate:"required,min=1,dive,required"`
}

var sampleApplyFulfillmentRecommendationsRequest = &ApplyFulfillmentRecommendationsRequest{
	ItemIDs: []string{apiresource.SampleItemID},
}

func (*ApplyFulfillmentRecommendationsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleApplyFulfillmentRecommendationsRequest)
}

// Adopts the recommended fulfillment policy for the named items, writing it as a per-item planning override.
//
// The recommendation is recomputed as part of applying it, rather than taken from the request. Advice read minutes ago may no longer be the advice — demand moves — and writing a stale verdict would set a policy the engine would not give today. What comes back is what was actually written.
//
// Takes effect on the next generated schedule; versions already generated keep the assumptions they were solved under.
type ApplyFulfillmentRecommendationsEndpoint struct{}

func (e *ApplyFulfillmentRecommendationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ApplyFulfillmentRecommendationsRequest, *apiresource.List[apiresource.FulfillmentRecommendation]] {
	return (&apiendpoint.APIEndpoint[*ApplyFulfillmentRecommendationsRequest, *apiresource.List[apiresource.FulfillmentRecommendation]]{
		Title:             "Apply Fulfillment Recommendations",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/fulfillment-recommendations/actions/apply",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeFulfillmentRecommendation,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ApplyFulfillmentRecommendationsRequest) (*apiresource.List[apiresource.FulfillmentRecommendation], *apierror.APIError) {
			return svc.(ProductionScheduleSettingsSvc).ApplyFulfillmentRecommendations
		},
	})
}
