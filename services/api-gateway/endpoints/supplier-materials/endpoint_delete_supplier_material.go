package suppliermaterialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a supplier material.
type DeleteSupplierMaterialRequest struct {
	// Supplier ID.
	SupplierID string `path:"supplier_id" validate:"required"`
	// Supplier material ID.
	MaterialID string `path:"id" validate:"required"`
}

// Deletes a supplier material association.
type DeleteSupplierMaterialEndpoint struct{}

func (e *DeleteSupplierMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSupplierMaterialRequest, *apiresource.SupplierMaterial] {
	return (&apiendpoint.APIEndpoint[*DeleteSupplierMaterialRequest, *apiresource.SupplierMaterial]{
		Title:             "Delete Supplier Material",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/suppliers/{supplier_id}/materials/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError) {
			return svc.(SupplierMaterialSvc).DeleteSupplierMaterial
		},
	})
}
