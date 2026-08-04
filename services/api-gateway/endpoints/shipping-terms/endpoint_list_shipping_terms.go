package shippingtermep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list shipping terms.
type ListShippingTermsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of shipping terms, newest first.
//
// Both the terms your account has created and the system-provided default terms are returned. The `q` parameter matches on the shipping term name.
type ListShippingTermsEndpoint struct{}

func (e *ListShippingTermsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListShippingTermsRequest, *apiresource.List[apiresource.ShippingTerm]] {
	return (&apiendpoint.APIEndpoint[*ListShippingTermsRequest, *apiresource.List[apiresource.ShippingTerm]]{
		Title:               "List Shipping Terms",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/operations/shipping-terms",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShippingTerms, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeShippingTerm,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingTerm,
			Fields:     []string{"owner", "owner.account", "flat_rate.unit", "minimum_order_value.unit", "free_shipping_service_levels"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListShippingTermsRequest) (*apiresource.List[apiresource.ShippingTerm], *apierror.APIError) {
			return svc.(ShippingTermSvc).ListShippingTerms
		},
	})
}
