package billingep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Returns the pricing plans an account can sign up for, with their limits and marketing copy.
//
// Only publicly listed plans that are currently in effect are returned, so privately negotiated and retired plans never appear here.
type GetPricingPlansEndpoint struct{}

func (e *GetPricingPlansEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPricingPlansRequest, *apiresource.List[apiresource.PricingPlan]] {
	return (&apiendpoint.APIEndpoint[*ListPricingPlansRequest, *apiresource.List[apiresource.PricingPlan]]{
		Title:             "List Pricing Plans",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/billing/plans",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypePricingPlan,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPricingPlansRequest) (*apiresource.List[apiresource.PricingPlan], *apierror.APIError) {
			return svc.(BillingSvc).GetPricingPlans
		},
	})
}

// ListPricingPlansRequest pages through the pricing plans.
type ListPricingPlansRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}
