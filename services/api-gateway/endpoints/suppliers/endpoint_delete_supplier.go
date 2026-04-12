package supplierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteSupplierRequest is the request to delete a single supplier.
type DeleteSupplierRequest struct {
	// The ID of the supplier to delete.
	SupplierID string `path:"id" validate:"required"`
}

type DeleteSupplierEndpoint struct{}

func (e *DeleteSupplierEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSupplierRequest, *apiresource.SupplierDetail] {
	return &apiendpoint.APIEndpoint[*DeleteSupplierRequest, *apiresource.SupplierDetail]{
		Title:             "Delete Supplier",
		Description:       "Deletes a supplier and its associated account relations, addresses, and account users.",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/suppliers/{id}",
		Request:           &DeleteSupplierRequest{},
		Response:          &apiresource.SupplierDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError) {
			return svc.(SupplierSvc).DeleteSupplier
		},
	}
}
