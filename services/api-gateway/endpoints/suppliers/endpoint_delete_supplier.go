package supplierep

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

// Request to delete a supplier.
type DeleteSupplierRequest struct {
	// Supplier ID.
	SupplierID string `path:"id" validate:"required"`
}

// Deletes a supplier.
//
// Only while no purchase order or supplier item references it. Its addresses and users are deleted with it. Archive a supplier that's in use by setting `status` to `archived`.
type DeleteSupplierEndpoint struct{}

func (e *DeleteSupplierEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSupplierRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteSupplierRequest, *object.Deleted]{
		Title:             "Delete Supplier",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/suppliers/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeSupplier,
		// Single delete checks suppliers:update downstream (Dashboard convention), not suppliers:delete.
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSupplierRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(SupplierSvc).DeleteSupplier, constants.ObjectTypeSupplier, "id")
		},
	})
}
