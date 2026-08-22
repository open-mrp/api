package productionschedulesettingsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list fulfillment recommendations.
type ListFulfillmentRecommendationsRequest struct{}

// Returns, for every sellable SKU, whether it should be built to stock or only against orders — and the measurement that decided.
//
// The rules are ordered and the first match wins. Lead-time feasibility is checked before anything else: if customers are promised less time than production needs, building to order is not possible rather than not preferred, and no amount of lumpy demand changes that. After that the engine looks for dead stock, a single contract customer, demand too erratic for a buffer to size, and slow-moving expensive units.
//
// Every verdict carries its numbers — demand interval, variability, customer concentration, promised lead time, annual cost of goods — so a planner can disagree with the rule rather than only with the answer. Thresholds are merchant-editable in the planning settings.
//
// Computed fresh on every call rather than stored. A recommendation is only meaningful next to current demand, and a saved one would go quietly stale; the durable artifact is the item setting written when someone agrees with it. Nothing here changes a plan on its own.
type ListFulfillmentRecommendationsEndpoint struct{}

func (e *ListFulfillmentRecommendationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListFulfillmentRecommendationsRequest, *apiresource.List[apiresource.FulfillmentRecommendation]] {
	return (&apiendpoint.APIEndpoint[*ListFulfillmentRecommendationsRequest, *apiresource.List[apiresource.FulfillmentRecommendation]]{
		Title:             "List Fulfillment Recommendations",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/fulfillment-recommendations",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeFulfillmentRecommendation,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListFulfillmentRecommendationsRequest) (*apiresource.List[apiresource.FulfillmentRecommendation], *apierror.APIError) {
			return svc.(ProductionScheduleSettingsSvc).ListFulfillmentRecommendations
		},
	})
}
