package shippingtermep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list shipping terms.
type ListShippingTermsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Returns a paginated list of shipping terms, newest first.
//
// Both the terms your account has created and the system-provided default terms are returned. The `q` parameter matches on the shipping term name.
type ListShippingTermsEndpoint struct{}

func (e *ListShippingTermsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListShippingTermsRequest, *apiresource.List[apiresource.ShippingTerm]] {
	return (&apiendpoint.APIEndpoint[*ListShippingTermsRequest, *apiresource.List[apiresource.ShippingTerm]]{
		Title:             "List Shipping Terms",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipping-terms",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainShippingTerms, Action: types.ActionRead}}},
		Preview:           true,
		ObjectType:        constants.ObjectTypeShippingTerm,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingTerm,
			Fields:     []string{"owner", "owner.account", "flat_rate.unit", "minimum_order_value.unit", "free_shipping_service_levels"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListShippingTermsRequest) (*apiresource.List[apiresource.ShippingTerm], *apierror.APIError) {
			return svc.(ShippingTermSvc).ListShippingTerms
		},
	})
}
