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
	// Case-insensitive substring match on `name`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return shipping terms with one of these statuses. Omit to return all statuses.
	Statuses []constants.ArchiveStatus `query:"statuses"`
}

// Returns a list of shipping terms.
//
// Includes the terms your account created and the system defaults every account shares. Terms are sorted by creation date, newest first.
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
		ObjectType:        constants.ObjectTypeShippingTerm,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingTerm,
			Fields:     []string{"owner", "owner.account", "free_shipping.service_levels"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListShippingTermsRequest) (*apiresource.List[apiresource.ShippingTerm], *apierror.APIError) {
			return svc.(ShippingTermSvc).ListShippingTerms
		},
	})
}
