package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Returns the pricing plans an account can sign up for, with their limits and marketing copy.
//
// Only publicly listed plans that are currently in effect are returned, so privately negotiated and retired plans never appear here.
type GetPricingPlansEndpoint struct{}

func (e *GetPricingPlansEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.PaginationRequest, *apiresource.List[apiresource.PricingPlan]] {
	return (&apiendpoint.APIEndpoint[*apiresource.PaginationRequest, *apiresource.List[apiresource.PricingPlan]]{
		Title:             "List Pricing Plans",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/billing/plans",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePricingPlan,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.PaginationRequest) (*apiresource.List[apiresource.PricingPlan], *apierror.APIError) {
			return svc.(BillingSvc).GetPricingPlans
		},
	})
}
