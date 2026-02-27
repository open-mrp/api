package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

const getPricingPlansDescription string = `Returns a paginated list of available pricing plans with their limits and features.`

type GetPricingPlansEndpoint struct{}

func (e *GetPricingPlansEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.PaginationRequest, *apiresource.List[apiresource.PricingPlan]] {
	return &apiendpoint.APIEndpoint[*apiresource.PaginationRequest, *apiresource.List[apiresource.PricingPlan]]{
		Title:             "List Pricing Plans",
		Description:       getPricingPlansDescription,
		Method:            http.MethodGet,
		Route:             "/v1/billing/plans",
		Request:           &apiresource.PaginationRequest{},
		Response:          &apiresource.List[apiresource.PricingPlan]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.PaginationRequest) (*apiresource.List[apiresource.PricingPlan], *apierror.APIError) {
			return svc.(BillingSvc).GetPricingPlans
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
