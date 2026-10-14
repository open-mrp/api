package shippingcaseep

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

// Request to delete a shipping case.
type DeleteShippingCaseRequest struct {
	// Shipping case ID.
	ShippingCaseID string `path:"id" validate:"required"`
}

// Permanently deletes a shipping case.
//
// Only the case is removed; its shipment, the shipment's lines, and the shipment's other cases are left untouched. Deleting a case that has already been deleted returns an error rather than succeeding again. A case whose shipping label was bought cannot be deleted until its shipment is voided, which refunds the label; a conflict error is returned instead.
type DeleteShippingCaseEndpoint struct{}

func (e *DeleteShippingCaseEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteShippingCaseRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteShippingCaseRequest, *object.Deleted]{
		Title:             "Delete Shipping Case",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/shipping-cases/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeShippingCase,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		// DeleteShippingCase enforces shipments:delete in the service (shipping cases
		// are a facet of shipments). Declared here to match that enforcement.
		Auth: authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteShippingCaseRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ShippingCaseSvc).DeleteShippingCase, constants.ObjectTypeShippingCase, "id")
		},
	})
}
