package supplierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteSupplierRequest is the request to delete a supplier.
type DeleteSupplierRequest struct {
	// Supplier ID.
	SupplierID string `path:"id" validate:"required"`
}

// Deletes a supplier and its associated account relations, addresses, and account users.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError) {
			return svc.(SupplierSvc).DeleteSupplier
		},
	})
}
