package shippingtermep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to retrieve a shipping term.
type RetrieveShippingTermRequest struct {
	// Shipping term ID.
	ShippingTermID string `path:"id" validate:"required"`
}

// Returns a shipping term by ID.
type RetrieveShippingTermEndpoint struct{}

func (e *RetrieveShippingTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveShippingTermRequest, *apiresource.ShippingTerm] {
	return (&apiendpoint.APIEndpoint[*RetrieveShippingTermRequest, *apiresource.ShippingTerm]{
		Title:             "Retrieve Shipping Term",
		Method:            http.MethodGet,
		Route:             "/v1/operations/shipping-terms/{id}",
		ContentType:       "application/json",
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError) {
			return svc.(ShippingTermSvc).GetShippingTerm
		},
	})
}
