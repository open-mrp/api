package shippingcaseep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a shipping case.
type DeleteShippingCaseRequest struct {
	// Shipping case ID.
	ShippingCaseID string `path:"id" validate:"required"`
}

// Permanently deletes a shipping case.
//
// Only the case is removed; its shipment, the shipment's lines, and the shipment's other cases are left untouched. Deleting a case that has already been deleted returns an error rather than succeeding again.
type DeleteShippingCaseEndpoint struct{}

func (e *DeleteShippingCaseEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteShippingCaseRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteShippingCaseRequest, *apiresource.EmptyResource]{
		Title:             "Delete Shipping Case",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/shipping-cases/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		// DeleteShippingCase enforces shipments:delete in the service (shipping cases
		// are a facet of shipments). Declared here to match that enforcement.
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteShippingCaseRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ShippingCaseSvc).DeleteShippingCase
		},
	})
}
