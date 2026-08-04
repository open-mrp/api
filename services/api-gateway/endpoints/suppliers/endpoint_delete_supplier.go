package supplierep

import (
	"context"
	"net/http"

	"github.com/augno/api/services/auth-service/pkg/types"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a supplier.
type DeleteSupplierRequest struct {
	// Supplier ID.
	SupplierID string `path:"id" validate:"required"`
}

// Deletes a supplier.
//
// The supplier's saved addresses and any users belonging to the supplier are deleted along with it. Returns the supplier as it looked immediately before deletion. Deleting a supplier that has already been deleted returns an error rather than succeeding again.
type DeleteSupplierEndpoint struct{}

func (e *DeleteSupplierEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSupplierRequest, *apiresource.SupplierDetail] {
	return (&apiendpoint.APIEndpoint[*DeleteSupplierRequest, *apiresource.SupplierDetail]{
		Title:             "Delete Supplier",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/suppliers/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		// Single delete checks suppliers:update downstream (Dashboard convention), not suppliers:delete.
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError) {
			return svc.(SupplierSvc).DeleteSupplier
		},
	})
}
