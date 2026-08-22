package shippingtermep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete a shipping term.
type DeleteShippingTermRequest struct {
	// Shipping term ID.
	ShippingTermID string `path:"id" validate:"required"`
}

// Deletes a shipping term owned by your account.
//
// System-provided default shipping terms cannot be deleted. The term's free-shipping service level rules, flat rate and minimum order value go with it, and deleting a term that has already been deleted returns an error rather than succeeding again.
type DeleteShippingTermEndpoint struct{}

func (e *DeleteShippingTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteShippingTermRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteShippingTermRequest, *apiresource.EmptyResource]{
		Title:               "Delete Shipping Term",
		Method:              http.MethodDelete,
		Route:               "/v1/operations/shipping-terms/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShippingTerms, Action: types.ActionDelete}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteShippingTermRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ShippingTermSvc).DeleteShippingTerm
		},
	})
}
