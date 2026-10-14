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
	"github.com/open-mrp/apikit/object"
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

func (e *DeleteShippingTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteShippingTermRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteShippingTermRequest, *object.Deleted]{
		Title:             "Delete Shipping Term",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/shipping-terms/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeShippingTerm,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainShippingTerms, Action: types.ActionDelete}}},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteShippingTermRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ShippingTermSvc).DeleteShippingTerm, constants.ObjectTypeShippingTerm, "id")
		},
	})
}
